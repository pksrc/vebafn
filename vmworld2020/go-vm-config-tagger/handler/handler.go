package function

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"

	handler "github.com/openfaas/templates-sdk/go-http"
	"github.com/pelletier/go-toml"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"d
)

const secretPath = "/var/openfaas/secrets/vcconfig"

// vcConfig represents the toml vcconfig file
type vcConfig struct {
	VCenter struct {
		Server   string
		User     string
		Password string
		Insecure bool
	}
}

// vsClient stores vSphere connection information.
type vsClient struct {
	govmomi *govmomi.Client
	rest    *rest.Client
	tagMgr  *tags.Manager
}

// cloudEvent stores incoming event data.
type cloudEvent struct {
	Data types.AlarmStatusChangedEvent
}

// Handle a function invocation
func Handle(req handler.Request) (handler.Response, error) {
	ctx := context.Background()

	cloudEvt, err := parseCloudEvent(req.Body)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("parsing cloud event data: %w", err))
	}

	// Determine if data AlarmStatusChangedEvent is correct.
	if !isCpuOrMemoryAlarm(cloudEvt) {
		message := "Alert not for CPU/Memory in red, nothing to do."
		log.Println(message)

		return handler.Response{
			Body:       []byte(message),
			StatusCode: http.StatusOK,
		}, nil
	}

	// Load config every time, to ensure the most updated version is used.
	cfg, err := loadTomlCfg(secretPath)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("loading of vcconfig: %w", err))
	}

	vsClient, err := newClient(ctx, cfg)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("connecting to vSphere: %w", err))
	}

	// Retrieve the Managed Object Reference from the event.
	vmMOR, err := eventVmMoRef(cloudEvt)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("retrieving VM managed reference object: %w", err))
	}

	// moVM contains the memory and CPU config values.
	moVM, err := vsClient.moVirtualMachine(ctx, vmMOR)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("getting vm configs: %w", err))
	}

	catID, tagID, err := vsClient.findIncrementedTag(ctx, cloudEvt, moVM)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("finding incremented tag: %w", err))
	}

	message := "No tag to attach."

	if tagID != "" {
		// Detach tags in the same catID, but different tagID.
		err = vsClient.detachTags(ctx, catID, tagID, vmMOR)
		if err != nil {
			return errRespondAndLog(fmt.Errorf("detaching old tag(s): %w", err))
		}

		err = vsClient.tagMgr.AttachTag(ctx, tagID, vmMOR)
		if err != nil {
			return errRespondAndLog(fmt.Errorf("tagging managed reference object: %w", err))
		}

		message = fmt.Sprintf("Attached tag %v.\n", tagID)
	}

	log.Println(message)

	return handler.Response{
		Body:       []byte(message),
		StatusCode: http.StatusOK,
	}, nil
}

func errRespondAndLog(err error) (handler.Response, error) {
	if debug() {
		log.Println(err.Error())
	}

	return handler.Response{
		Body:       []byte(err.Error()),
		StatusCode: http.StatusInternalServerError,
	}, err
}

// Debug determines verbose logging
func debug() bool {
	verbose := os.Getenv("write_debug")

	if verbose == "true" {
		return true
	}

	return false
}

func parseCloudEvent(req []byte) (cloudEvent, error) {
	var event cloudEvent

	err := json.Unmarshal(req, &event)
	if err != nil {
		return cloudEvent{}, fmt.Errorf("unmarshalling json: %w", err)
	}

	if err := isValidEvent(event); err != nil {
		return cloudEvent{}, err
	}

	return event, nil
}

func isValidEvent(event cloudEvent) error {
	if event.Data.Vm == nil || event.Data.Vm.Vm.Value == "" {
		return errors.New("empty VM managed object reference")
	}

	if event.Data.Alarm.Name == "" || event.Data.To == "" {
		return errors.New("insufficent alarm infomration")
	}

	return nil
}

func isCpuOrMemoryAlarm(event cloudEvent) bool {
	alarm := false

	if event.Data.To == "red" && (event.Data.Alarm.Name == "VM Memory Usage" || event.Data.Alarm.Name == "VM CPU Usage") {
		alarm = true
	}

	return alarm
}

func loadTomlCfg(path string) (*vcConfig, error) {
	var cfg vcConfig

	secret, err := toml.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading vcconfig.toml: %w", err)
	}

	err = secret.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling vcconfig.toml: %w", err)
	}

	err = validateConfig(cfg)
	if err != nil {
		return nil, err
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
		govmomi: gc,
		rest:    rc,
		tagMgr:  tm,
	}

	err = vsc.rest.Login(ctx, u.User)
	if err != nil {
		return nil, fmt.Errorf("logging into rest api: %w", err)
	}

	return &vsc, nil
}

func eventVmMoRef(event cloudEvent) (types.ManagedObjectReference, error) {
	// Fill information in the request into a govmomi type.
	moRef := types.ManagedObjectReference{
		Type:  event.Data.Vm.Vm.Type,
		Value: event.Data.Vm.Vm.Value,
	}

	return moRef, nil
}

// unappliedConfigs returns configurations that are not current.
func (c *vsClient) moVirtualMachine(ctx context.Context, mor types.ManagedObjectReference) (mo.VirtualMachine, error) {
	// Look for current hardware configuration
	var moVM mo.VirtualMachine

	pc := property.DefaultCollector(c.govmomi.Client)

	err := pc.Retrieve(ctx, []types.ManagedObjectReference{mor}, []string{}, &moVM)
	if err != nil {
		return mo.VirtualMachine{}, err
	}

	if moVM.Config == nil {
		return mo.VirtualMachine{}, errors.New("managed object VM Config is empty")
	}

	if moVM.Config.Hardware.NumCPU == 0 || moVM.Config.Hardware.MemoryMB == 0 {
		return mo.VirtualMachine{}, errors.New("managed object VM missing CPU and/or Memory info")
	}

	return moVM, nil
}

// findIncrementedTag finds the current config value for the type, and will select
// the tag that is an increment above it (but below the limits).
func (clt *vsClient) findIncrementedTag(ctx context.Context, ce cloudEvent, moVM mo.VirtualMachine) (string, string, error) {
	catName := catName(ce.Data.Alarm.Name)
	tagName := ""
	// get the expected name of the tag (incremented value)

	switch catName {
	case "config.hardware.numCPU":
		// CPU tags are easy. Just increment it up to the max of 4.
		tagName = incCpuVal(int(moVM.Config.Hardware.NumCPU))
	case "config.hardware.memoryMB":
		// Mem tags are a bit tricky. Gotta find the exponent to the 2 base.
		// Then, increment the exponent up to the max of 23 (8 gb RAM).
		tagName = incMemVal(float64(moVM.Config.Hardware.MemoryMB))
	}

	tagList, err := clt.tagMgr.GetTagsForCategory(ctx, catName)
	if err != nil {
		return "", "", err
	}

	catID, tagID := findCatAndTagIDs(tagList, tagName)

	// return the tag ID given the name.
	return catID, tagID, nil
}

// catName returns the category name based on alarm name.
func catName(alarmName string) string {
	switch alarmName {
	case "VM CPU Usage":
		return "config.hardware.numCPU"
	case "VM Memory Usage":
		return "config.hardware.memoryMB"
	}

	return ""
}

func incCpuVal(numCPU int) string {
	newNum := 4
	numCPU++
	if numCPU < newNum {
		newNum = numCPU
	}

	log.Printf("\ncurrent CPU: %v, new CPU: %v\n", numCPU, newNum)
	return strconv.Itoa(newNum)
}

// Use MB values, not bytes.
func incMemVal(mem float64) string {
	// 2^13 = 8192. 8GB is max RAM.
	maxExp := 13
	newMem := 1 << maxExp

	exp := int(math.Round(math.Log10(mem) / math.Log10(2)))
	exp++

	if exp < maxExp {
		newMem = 1 << exp
	}

	log.Printf("\ncurrent memory: %v, new memory: %v\n", mem, newMem)
	return strconv.Itoa(newMem)
}

func findCatAndTagIDs(ts []tags.Tag, tn string) (string, string) {
	for _, t := range ts {
		if t.Name == tn {
			return t.CategoryID, t.ID
		}
	}

	return "", ""
}

func (clt *vsClient) detachTags(ctx context.Context, catID, tagID string, mor types.ManagedObjectReference) error {
	tagList, err := clt.tagMgr.GetAttachedTags(ctx, mor)
	if err != nil {
		return err
	}

	// Loop through the tags and detach the ones that are in catID but are not tagID.
	for _, t := range tagList {
		if t.CategoryID == catID && t.ID != tagID {
			if err := clt.tagMgr.DetachTag(ctx, t.ID, mor); err != nil {
				return err
			}
		}
	}

	return nil
}
