package main

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
)

type vsClient struct {
	govmomi    *govmomi.Client
	rest       *rest.Client
	tagManager *tags.Manager
}

type vcConfig struct {
	server   string
	user     string
	password string
	insecure bool
}

func main() {
	ctx := context.Background()

	cfg, err := vcCredentials()
	if err != nil {
		fmt.Println(err)
		return
	}

	vsClt, err := newClient(ctx, cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	// These are config.hardware properties listed in vim25/types/types.go
	// vSphere API Doc: https://vdc-download.vmware.com/vmwb-repository/dcr-public/b50dcbbf-051d-4204-a3e7-e1b618c1e384/538cf2ec-b34f-4bae-a332-3820ef9e7773/vim.vm.VirtualHardware.html
	hwProps := []string{
		"numCPU",
		"memoryMB",
		"numCoresPerSocket",
		"memoryHotAddEnabled",
		"cpuHotRemoveEnabled",
		"cpuHotAddEnabled",
	}

	cfgTags, err := userSelectTags(hwProps)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err = makeTags(ctx, vsClt, cfgTags); err != nil {
		fmt.Println(err)
		return
	}
}

// TODO: if tag or category already exists, skip.
func makeTags(ctx context.Context, vsc *vsClient, cfgTags map[string][]string) error {

	for t := range cfgTags {
		preDesc := "Hardware configuration for "
		preName := "config.hardware."

		if t == "memoryHotAddEnabled" || t == "cpuHotRemoveEnabled" || t == "cpuHotAddEnabled" {
			preDesc = "Configuration for "
			preName = "config."
		}

		cID, err := vsc.tagManager.CreateCategory(ctx, &tags.Category{
			AssociableTypes: []string{"VirtualMachine"},
			Cardinality:     "SINGLE",
			Description:     preDesc,
			Name:            preName + t,
		})
		if err != nil {
			return err
		}

		for _, p := range cfgTags[t] {
			fmt.Println("Creating " + p + " tag for " + t)
			_, err := vsc.tagManager.CreateTag(ctx, &tags.Tag{
				CategoryID:  cID,
				Description: "Preset for " + t + " configuration",
				Name:        p,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Ask user to choose which tags to generate.
func userSelectTags(props []string) (map[string][]string, error) {
	fmt.Printf("There are %d properties from which tags can be created. The properties are %+v.\n", len(props), props)
	fmt.Printf("For each property, please indicate whether or not you want to create tags.\n")

	cfgTags := make(map[string][]string)
	reader := bufio.NewReader(os.Stdin)

	// Ask user which of the properties should be made to tags.
	for _, p := range props {
		fmt.Printf("Create tags for %s? Y/n\n", p)

		userInput, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		if userInput == "Y\n" || userInput == "y\n" {
			ps := setPresets(p)
			cfgTags[p] = ps
		}
	}

	fmt.Println("Tags to be created for ", cfgTags)

	return cfgTags, nil
}

// TODO: allow users to set their own presets through yaml or csv files.
func setPresets(prop string) []string {

	switch prop {
	case "numCPU":
		return []string{"1", "2", "3", "4"}
	case "memoryMB":
		return []string{"1024", "2048", "4096", "8192", "16384"}
	case "numCoresPerSocket":
		return []string{"1", "2", "3", "4"}
	case "memoryHotAddEnabled":
		return []string{"true", "false"}
	case "cpuHotRemoveEnabled":
		return []string{"true", "false"}
	case "cpuHotAddEnabled":
		return []string{"true", "false"}
	}
	return []string{}
}

// newClient connects to vSphere govmomi API
func newClient(ctx context.Context, cfg vcConfig) (*vsClient, error) {
	u := url.URL{
		Scheme: "https",
		Host:   cfg.server,
		Path:   "sdk",
	}

	u.User = url.UserPassword(cfg.user, cfg.password)
	insecure := cfg.insecure

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

// TODO: validate user input for vSphere host and credentials.
func vcCredentials() (vcConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("If credentials already set via environmental variables, press enter.")
	fmt.Println("What is the vSphere server address, e.g. 10.152.128.165:443?")
	server, err := reader.ReadString('\n')
	server = strings.TrimSuffix(server, "\n")
	if err != nil {
		return vcConfig{}, err
	}

	if server == "" {
		fmt.Print("Using environmental variable.\n\n")
		server = os.Getenv("VEBA_TAG_GEN_SERVER")
	}

	fmt.Println("vSphere username, e.g. Administrator")
	user, err := reader.ReadString('\n')
	user = strings.TrimSuffix(user, "\n")
	if err != nil {
		return vcConfig{}, err
	}

	if user == "" {
		fmt.Print("Using environmental variable.\n\n")
		user = os.Getenv("VEBA_TAG_GEN_USER")
	}

	fmt.Println("vSphere password")
	pass, err := reader.ReadString('\n')
	pass = strings.TrimSuffix(pass, "\n")
	if err != nil {
		return vcConfig{}, err
	}

	if pass == "" {
		fmt.Print("Using environmental variable.\n\n")
		pass = os.Getenv("VEBA_TAG_GEN_PASS")
	}

	cfg := vcConfig{server, user, pass, true}

	if pass == "" || user == "" || server == "" {
		fmt.Println("Unable to proceed without credentials.")
		return vcConfig{}, fmt.Errorf("pass set: %v, user set: %v, server set: %v", pass != "", user != "", server != "")
	}

	fmt.Println("Credentials have been set.")
	return cfg, nil
}
