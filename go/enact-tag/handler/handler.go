package function

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	handler "github.com/openfaas/templates-sdk/go-http"
	"github.com/pelletier/go-toml"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

const cfgPath = "/var/openfaas/secrets/vcconfig"

// vcConfig represents the toml vcconfig file
type vcConfig struct {
	VCenter struct {
		Server   string
		User     string
		Password string
		Insecure bool
	}
}

// Incoming is a subsection of a Cloud Event.
type incoming struct {
	Data types.Event `json:"data,omitempty"`
}

type vsClient struct {
	govmomi    *govmomi.Client
	rest       *rest.Client
	tagManager *tags.Manager
}

type vmConfig struct {
	name    string
	valBool bool
	valInt  int
}

type tagInfo struct {
	catID string
	tagID string
}

// Handle a function invocation
func Handle(req handler.Request) (handler.Response, error) {
	ctx := context.Background()

	// Load config every time, to ensure the most updated version is used.
	cfg, err := loadTomlCfg(cfgPath)
	if err != nil {
		wrapErr := fmt.Errorf("loading of vcconfig: %w", err)
		log.Println(wrapErr.Error())

		return handler.Response{
			Body:       []byte(wrapErr.Error()),
			StatusCode: http.StatusInternalServerError,
		}, wrapErr
	}

	vsClt, err := newClient(ctx, cfg)
	if err != nil {
		wrapErr := fmt.Errorf("connecting to vSphere: %w", err)
		log.Println(wrapErr.Error())

		return handler.Response{
			Body:       []byte(wrapErr.Error()),
			StatusCode: http.StatusInternalServerError,
		}, wrapErr
	}

	// The Mananged Object Reference for the VM that powered on.
	vmMoRef, err := eventVmMoRef(req.Body)
	if err != nil {
		wrapErr := fmt.Errorf("retrieving VM object: %w", err)
		log.Println(wrapErr.Error())

		return handler.Response{
			Body:       []byte(wrapErr.Error()),
			StatusCode: http.StatusInternalServerError,
		}, wrapErr
	}

	// Look for configurations that need to be enacted on VM.
	hwCfgs, err := vsClt.selectConfigs(ctx, vmMoRef)
	if err != nil {
		wrapErr := fmt.Errorf("retrieving config tags: %w", err)
		log.Println(wrapErr.Error())

		return handler.Response{
			Body:       []byte(wrapErr.Error()),
			StatusCode: http.StatusInternalServerError,
		}, wrapErr
	}

	var task *object.Task

	if len(hwCfgs) > 0 {
		task, err = vsClt.applyCfgs(ctx, *vmMoRef, hwCfgs)
		if err != nil {
			wrapErr := fmt.Errorf("applying config tags: %w", err)
			log.Println(wrapErr.Error())

			return handler.Response{
				Body:       []byte(wrapErr.Error()),
				StatusCode: http.StatusInternalServerError,
			}, wrapErr
		}
	}

	message := appliedConfigsMessage(task, hwCfgs)
	log.Println(message)

	return handler.Response{
		Body:       []byte(message),
		StatusCode: http.StatusOK,
	}, nil
}

func loadTomlCfg(path string) (*vcConfig, error) {
	var cfg vcConfig

	secret, err := toml.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to load vcconfig.toml: %w", err)
	}

	err = secret.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal vcconfig.toml: %w", err)
	}

	err = validateConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("insufficient information in vcconfig.toml: %w", err)
	}

	return &cfg, nil
}

// ValidateConfig ensures the bare minimum of information is in the config file.
func validateConfig(cfg vcConfig) error {
	reqFields := map[string]string{
		"vcenter server":   cfg.VCenter.Server,
		"vcenter user":     cfg.VCenter.User,
		"vcenter password": cfg.VCenter.Password,
	}

	// Multiple fields may be missing, but err on the first encountered.
	for k, v := range reqFields {
		if v == "" {
			return errors.New("required field(s) missing, including " + k)
		}
	}

	return nil
}

// newClient connects to vSphere govmomi API
func newClient(ctx context.Context, cfg *vcConfig) (*vsClient, error) {
	u := url.URL{
		Scheme: "https",
		Host:   cfg.VCenter.Server,
		Path:   "sdk",
	}

	u.User = url.UserPassword(cfg.VCenter.User, cfg.VCenter.Password)
	insecure := cfg.VCenter.Insecure

	gc, err := govmomi.NewClient(ctx, &u, insecure)
	if err != nil {
		return nil, fmt.Errorf("connecting to vSphere API: %w", err)
	}

	rc := rest.NewClient(gc.Client)
	tm := tags.NewManager(rc)

	vsc := vsClient{
		govmomi:    gc,
		rest:       rc,
		tagManager: tm,
	}

	err = vsc.rest.Login(ctx, u.User)
	if err != nil {
		return nil, fmt.Errorf("logging into rest api: %w", err)
	}

	return &vsc, nil
}

func eventVmMoRef(req []byte) (*types.ManagedObjectReference, error) {
	var event incoming
	var moRef types.ManagedObjectReference

	err := json.Unmarshal(req, &event)
	if err != nil {
		return nil, fmt.Errorf("parsing of request: %w", err)
	}

	if event.Data.Vm == nil || event.Data.Vm.Vm.Value == "" {
		return nil, errors.New("event object is not a VM")
	}

	// Fill information in the request into a govmomi type.
	moRef.Type = event.Data.Vm.Vm.Type
	moRef.Value = event.Data.Vm.Vm.Value

	return &moRef, nil
}

// Determine with the help of tags which hardware configurations and their values
// that need to be applied.
func (c *vsClient) selectConfigs(ctx context.Context, vmMoRef *types.ManagedObjectReference) ([]vmConfig, error) {

	// Tag objects attached to the VM.
	tagObjs, err := c.tagManager.GetAttachedTags(ctx, vmMoRef)
	if err != nil {
		return []vmConfig{}, fmt.Errorf("getting attached tags: %w", err)
	}

	tagInfo := tagInformation(tagObjs)

	// Get configs from tags.
	cfgs, err := c.configInfo(ctx, tagInfo)
	if err != nil {
		return []vmConfig{}, err
	}

	unappliedCfgs := c.unappliedConfigs(ctx, *vmMoRef, cfgs)

	return unappliedCfgs, nil
}

func tagInformation(tagObjs []tags.Tag) []tagInfo {
	info := make([]tagInfo, len(tagObjs))
	i := 0

	for _, t := range tagObjs {
		info[i] = tagInfo{
			tagID: t.ID,
			catID: t.CategoryID,
		}
		i++
	}

	return info
}

// Get only the hardware config tags.
func (c *vsClient) configInfo(ctx context.Context, info []tagInfo) ([]vmConfig, error) {
	hwTags := []vmConfig{}

	for _, i := range info {
		// Only select tag categories corresponding with configs we care about.
		hwProp, err := c.filteredConfigNames(ctx, i.catID)
		if err != nil {
			return []vmConfig{}, err
		}

		if hwProp == "" {
			continue
		}

		// Get the config value from the tag name.
		t, err := c.tagManager.GetTag(ctx, i.tagID)
		if err != nil {
			return []vmConfig{}, fmt.Errorf("getting tag from tagID: %w", err)
		}

		if hwProp == "numCPU" || hwProp == "memoryMB" || hwProp == "numCoresPerSocket" {
			val, err := strconv.Atoi(t.Name)
			if err != nil {
				return []vmConfig{}, fmt.Errorf("converting string to int: %w", err)
			}

			hwTags = append(hwTags, vmConfig{name: hwProp, valInt: val})
		}

		if hwProp == "memoryHotAddEnabled" || hwProp == "cpuHotRemoveEnabled" || hwProp == "cpuHotAddEnabled" {
			val, err := strconv.ParseBool(t.Name)
			if err != nil {
				return []vmConfig{}, fmt.Errorf("converting string to bool: %w", err)
			}

			hwTags = append(hwTags, vmConfig{name: hwProp, valBool: val})
		}
	}

	return hwTags, nil
}

// configName selects only category names that correspond with a VM config that we care about.
func (c *vsClient) filteredConfigNames(ctx context.Context, catID string) (string, error) {
	// These are the hardware configurations that we care to change.
	hwProps := []string{
		"config.hardware.numCPU",
		"config.hardware.memoryMB",
		"config.hardware.numCoresPerSocket",
		"config.memoryHotAddEnabled",
		"config.cpuHotRemoveEnabled",
		"config.cpuHotAddEnabled",
	}

	catObj, err := c.tagManager.GetCategory(ctx, catID)
	if err != nil {
		return "", fmt.Errorf("get category from catID: %w", err)
	}

	for _, p := range hwProps {
		if catObj.Name == p {
			temp := strings.TrimPrefix(catObj.Name, "config.")
			return strings.TrimPrefix(temp, "hardware."), nil
		}
	}

	return "", nil
}

// unappliedConfigs returns configurations that are not current.
func (c *vsClient) unappliedConfigs(ctx context.Context, moRef types.ManagedObjectReference, tagCfgs []vmConfig) []vmConfig {
	// Look for current hardware configuration
	var vm mo.VirtualMachine

	pc := property.DefaultCollector(c.govmomi.Client)
	pc.Retrieve(ctx, []types.ManagedObjectReference{moRef}, []string{}, &vm)

	currCfgs := currentConfigs(vm)

	// If the current hardware configuration matches what's in the tags,
	// remove them from the tags.
	unappliedTags := filterOutCurrentConfigs(tagCfgs, currCfgs)

	return unappliedTags
}

func currentConfigs(vm mo.VirtualMachine) []vmConfig {
	configs := make([]vmConfig, 6)

	configs[0] = vmConfig{name: "numCPU", valInt: int(vm.Config.Hardware.NumCPU)}
	configs[1] = vmConfig{name: "memoryMB", valInt: int(vm.Config.Hardware.MemoryMB)}
	configs[2] = vmConfig{name: "numCoresPerSocket", valInt: int(vm.Config.Hardware.NumCoresPerSocket)}

	if vm.Config.MemoryHotAddEnabled != nil {
		configs[3] = vmConfig{name: "memoryHotAddEnabled", valBool: *vm.Config.MemoryHotAddEnabled}
	}

	if vm.Config.CpuHotRemoveEnabled != nil {
		configs[4] = vmConfig{name: "cpuHotRemoveEnabled", valBool: *vm.Config.CpuHotRemoveEnabled}
	}

	if vm.Config.CpuHotAddEnabled != nil {
		configs[5] = vmConfig{name: "cpuHotAddEnabled", valBool: *vm.Config.CpuHotAddEnabled}
	}

	return configs
}

// filterOutCurrentConfigs removes the tag configs that are already current configs.
func filterOutCurrentConfigs(tagConfigs []vmConfig, currConfigs []vmConfig) []vmConfig {
	unappliedCfgs := []vmConfig{}

	for _, tc := range tagConfigs {
		if !isCfgCurrent(tc, currConfigs) {
			unappliedCfgs = append(unappliedCfgs, tc)
		}
	}
	return unappliedCfgs
}

// isHwCfgMatch determines if the given tag's hardware configuration
// is already the current configuration of the hardware.
func isCfgCurrent(tagConfig vmConfig, currConfigs []vmConfig) bool {
	for _, curr := range currConfigs {
		if tagConfig == curr {
			return true
		}
	}

	return false
}

// makeCfgsMatch sets configuration of the VM to that of the attached tag.
func (c *vsClient) applyCfgs(ctx context.Context, moRef types.ManagedObjectReference, cfgs []vmConfig) (*object.Task, error) {
	vm := object.NewVirtualMachine(c.govmomi.Client, moRef)
	desiredSpec := generateDesiredSpec(cfgs)

	task, err := vm.Reconfigure(ctx, desiredSpec)
	if err != nil {
		return nil, err
	}

	fmt.Println(task)
	return task, nil
}

func generateDesiredSpec(cfgs []vmConfig) types.VirtualMachineConfigSpec {
	var spec types.VirtualMachineConfigSpec

	for _, c := range cfgs {
		switch c.name {
		case "numCPU":
			spec.NumCPUs = int32(c.valInt)
		case "memoryMB":
			spec.MemoryMB = int64(c.valInt)
		case "numCoresPerSocket":
			spec.NumCoresPerSocket = int32(c.valInt)
		case "memoryHotAddEnabled":
			spec.MemoryHotAddEnabled = &c.valBool
		case "cpuHotRemoveEnabled":
			spec.CpuHotRemoveEnabled = &c.valBool
		case "cpuHotAddEnabled":
			spec.CpuHotAddEnabled = &c.valBool
		}
	}

	return spec
}

func appliedConfigsMessage(task *object.Task, cfgs []vmConfig) string {
	if task == nil {
		return "Nothing to configure."
	}

	msg := task.String() + " set "

	for _, c := range cfgs {
		if c.name == "memoryHotAddEnabled" || c.name == "cpuHotRemoveEnabled" || c.name == "cpuHotAddEnabled" {
			msg += fmt.Sprintf("%s to %v, ", c.name, c.valBool)
		} else {
			msg += fmt.Sprintf("%s to %v, ", c.name, c.valInt)
		}
	}

	return strings.TrimRight(msg, ", ") + "."
}
