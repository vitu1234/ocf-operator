package controllers

import (
	logging "log"

	"github.com/jessevdk/go-flags"
	iotv1alpha1 "github.com/vitu1234/ocf-operator/api/v1alpha1"
	OCFClient "github.com/vitu1234/ocf-operator/controllers/ocfclient"
)

// Declare global variables
var (
	opts       iotv1alpha1.Options
	ocf_client OCFClient.OCFClient
)

func init() {
	// Initialize the global variables
	// Initialize the global variables
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		logging.Println("Parsing command options has failed : " + err.Error())
	}

	OCFClient.ReadCommandOptions(opts)

	// Create OCF Client
	ocfClient := OCFClient.OCFClient{}
	err = ocfClient.Initialize()
	if err != nil {
		logging.Println("OCF Client has failed to initialize : " + err.Error())
	}

	// Assign the initialized variables to the global variables
	// opts = opts
	ocf_client = ocfClient

}
