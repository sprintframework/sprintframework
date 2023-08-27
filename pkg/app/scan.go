/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package app

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprintframework/pkg/assets"
	"github.com/sprintframework/sprintframework/pkg/assetsgz"
	"github.com/sprintframework/sprintframework/pkg/resources"
	"os"
)

var DefaultFileModes = map[string]interface{} {
	"log.dir": os.FileMode(0775),
	"log.file": os.FileMode(0664),
	"backup.file": os.FileMode(0664),
	"exe.file": os.FileMode(0775),
	"run.dir": os.FileMode(0775),
	"pid.file": os.FileMode(0666),
	"data.dir": os.FileMode(0770),
	"data.file": os.FileMode(0664),
}

var DefaultResources = &glue.ResourceSource{
	Name: "resources",
	AssetNames: resources.AssetNames(),
	AssetFiles: resources.AssetFile(),
}

var DefaultAssets = &glue.ResourceSource{
	Name: "assets",
	AssetNames: assets.AssetNames(),
	AssetFiles: assets.AssetFile(),
}

var DefaultGzipAssets = &glue.ResourceSource{
	Name: "assets-gzip",
	AssetNames: assetsgz.AssetNames(),
	AssetFiles: assetsgz.AssetFile(),
}

var DefaultApplicationBeans = []interface{}{
	ApplicationFlags(100000), // override any property resolvers
	FlagSetFactory(),
	ResourceService(),
}
