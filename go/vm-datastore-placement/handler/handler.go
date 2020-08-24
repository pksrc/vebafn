package function

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	handler "github.com/openfaas/templates-sdk/go-http"
	"github.com/pelletier/go-toml"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const secretPath = "/var/openfaas/secrets/vcconfigtoml"

// Handle a function invocation
func Handle(req handler.Request) (handler.Response, error) {
	ctx := context.Background()

	// Load config every time, to ensure the most updated version is used.
	cfg, err := loadTomlCfg(secretPath)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("loading of vcconfig: %w", err))
	}

	vsClt, err := newClient(ctx, cfg)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("connecting to vSphere: %w", err))
	}

	cloudEvt, err := parseCloudEvent(req.Body)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("parsing cloud event data: %w", err))
	}

	// Determine if data AlarmStatusChangedEvent is correct.
	if !isStorageInAlarm(cloudEvt) {
		message := "Storage not in red alert, nothing to do."
		log.Println(message)

		return handler.Response{
			Body:       []byte(message),
			StatusCode: http.StatusOK,
		}, nil
	}

	// The Mananged Object Reference for the VM that caused storage alarm.
	vmMOR, err := eventVmMoRef(cloudEvt)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("retrieving VM object: %w", err))
	}

	vm := object.NewVirtualMachine(vsClt.govmomi.Client, vmMOR)

	// TODO: Determine relocation spec without hardcoding.
	spec := generateRelocSpec()

	// Relocate the VM onto a different datastore.
	task, err := vm.Relocate(ctx, spec, types.VirtualMachineMovePriorityHighPriority)
	if err != nil {
		return errRespondAndLog(fmt.Errorf("connecting to vSphere: %w", err))
	}

	message := relocatedMessage(task)
	log.Println(message)

	return handler.Response{
		Body:       []byte(message),
		StatusCode: http.StatusOK,
	}, nil
}

func errRespondAndLog(err error) (handler.Response, error) {
	log.Println(err.Error())

	return handler.Response{
		Body:       []byte(err.Error()),
		StatusCode: http.StatusInternalServerError,
	}, err
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

// validateConfig ensures the bare minimum of information is in the config file.
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

	vsc := vsClient{
		govmomi: gc,
	}

	return &vsc, nil
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

func isStorageInAlarm(event cloudEvent) bool {
	alarm := false

	if event.Data.Alarm.Name == "VM Storage Usage" && event.Data.To == "red" {
		alarm = true
	}

	return alarm
}

func eventVmMoRef(event cloudEvent) (types.ManagedObjectReference, error) {
	// Fill information in the request into a govmomi type.
	moRef := types.ManagedObjectReference{
		Type:  event.Data.Vm.Vm.Type,
		Value: event.Data.Vm.Vm.Value,
	}

	return moRef, nil
}

// isValidEvent ensures the necessary information has been sent.
func isValidEvent(event cloudEvent) error {
	if event.Data.Vm == nil || event.Data.Vm.Vm.Value == "" {
		return errors.New("empty managed object reference")
	}

	if (event.Data.To == "" || event.Data.Alarm == types.AlarmEventArgument{}) {
		return errors.New("insufficient alarm information")
	}

	return nil
}

func generateRelocSpec() types.VirtualMachineRelocateSpec {
	// Resource pool managed object reference
	poolMOR := types.ManagedObjectReference{
		Type:  "ResourcePool",
		Value: "resgroup-1030",
	}

	// Host managed object reference
	hostMOR := types.ManagedObjectReference{
		Type:  "HostSystem",
		Value: "host-1031",
	}

	// Datastore managed object reference
	dsMOR := types.ManagedObjectReference{
		Type:  "Datastore",
		Value: "datastore-1032",
	}

	spec := types.VirtualMachineRelocateSpec{
		Host:         &hostMOR,
		Pool:         &poolMOR,
		Datastore:    &dsMOR,
		DiskMoveType: "moveAllDiskBackingsAndConsolidate",
	}

	return spec
}

func relocatedMessage(task *object.Task) string {
	if task == nil {
		return "Nothing relocated."
	}

	msg := task.String()

	return msg
}
