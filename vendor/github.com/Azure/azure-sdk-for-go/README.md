# Microsoft Azure SDK for Go

This project provides various Go packages to perform operations
on Microsoft Azure REST APIs.

[![GoDoc](https://godoc.org/github.com/Azure/azure-sdk-for-go?status.svg)](https://godoc.org/github.com/Azure/azure-sdk-for-go) [![Build Status](https://travis-ci.org/Azure/azure-sdk-for-go.svg?branch=master)](https://travis-ci.org/Azure/azure-sdk-for-go)

> **NOTE:** This repository is under heavy ongoing development and
is likely to break over time. We currently do not have any releases
yet. If you are planning to use the repository, please consider vendoring
the packages in your project and update them when a stable tag is out.

# Packages

## Azure Resource Manager (ARM)

[About ARM](/arm/README.md)

- [authorization](/arm/authorization)
- [batch](/arm/batch)
- [cdn](/arm/cdn)
- [cognitiveservices](/arm/cognitiveservices)
- [compute](/arm/compute)
- [containerservice](/arm/containerservice)
- [datalake-store](/arm/datalake-store)
- [devtestlabs](/arm/devtestlabs)
- [dns](/arm/dns)
- [intune](/arm/intune)
- [iothub](/arm/iothub)
- [keyvault](/arm/keyvault)
- [logic](/arm/logic)
- [machinelearning](/arm/machinelearning)
- [mediaservices](/arm/mediaservices)
- [mobileengagement](/arm/mobileengagement)
- [network](/arm/network)
- [notificationhubs](/arm/notificationhubs)
- [powerbiembedded](/arm/powerbiembedded)
- [redis](/arm/redis)
- [resources](/arm/resources)
- [scheduler](/arm/scheduler)
- [search](/arm/search)
- [servicebus](/arm/servicebus)
- [sql](/arm/sql)
- [storage](/arm/storage)
- [trafficmanager](/arm/trafficmanager)
- [web](/arm/web)

## Azure Service Management (ASM), aka classic deployment

[About ASM](/management/README.md)

- [affinitygroup](/management/affinitygroup)
- [hostedservice](/management/hostedservice)
- [location](/management/location)
- [networksecuritygroup](/management/networksecuritygroup)
- [osimage](/management/osimage)
- [sql](/management/sql)
- [storageservice](/management/storageservice)
- [virtualmachine](/management/virtualmachine)
- [virtualmachinedisk](/management/virtualmachinedisk)
- [virtualmachineimage](/management/virtualmachineimage)
- [virtualnetwork](/management/virtualnetwork)
- [vmutils](/management/vmutils)

## Azure Storage SDK for Go

[About Storage](/storage/README.md)

- [storage](/storage)

# Installation

- [Install Go 1.7](https://golang.org/dl/).

- Go get the SDK:

```
$ go get -d github.com/Azure/azure-sdk-for-go
```

> **IMPORTANT:** We highly suggest vendoring Azure SDK for Go as a dependency. For vendoring dependencies, Azure SDK for Go uses [glide](https://github.com/Masterminds/glide). If you haven't already, install glide. Navigate to your project directory and install the dependencies.

```
$ cd your/project
$ glide create
$ glide install
```

# Documentation

Read the Godoc of the repository at [Godoc.org](http://godoc.org/github.com/Azure/azure-sdk-for-go/).

# Contribute

If you would like to become an active contributor to this project please follow the instructions provided in [Microsoft Azure Projects Contribution Guidelines](http://azure.github.io/guidelines/).

# License

This project is published under [Apache 2.0 License](LICENSE).

-----
This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
