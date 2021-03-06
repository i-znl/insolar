//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/insolar/insolar/certificate"
	"github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/server"
)

type inputParams struct {
	configPath        string
	isGenesis         bool
	genesisConfigPath string
	genesisKeyOut     string
	traceEnabled      bool
}

func parseInputParams() inputParams {
	var rootCmd = &cobra.Command{Use: "insolard"}
	var result inputParams
	rootCmd.Flags().StringVarP(&result.configPath, "config", "c", "", "path to config file")
	rootCmd.Flags().StringVarP(&result.genesisConfigPath, "genesis", "g", "", "path to genesis config file")
	rootCmd.Flags().StringVarP(&result.genesisKeyOut, "keyout", "", ".", "genesis certificates path")
	rootCmd.Flags().BoolVarP(&result.traceEnabled, "trace", "t", false, "enable tracing")
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal("Wrong input params:", err)
	}

	if result.genesisConfigPath != "" {
		result.isGenesis = true
	}

	return result
}

func main() {
	params := parseInputParams()
	jww.SetStdoutThreshold(jww.LevelDebug)

	if params.isGenesis {
		s := server.NewGenesisServer(
			params.configPath,
			params.traceEnabled,
			params.genesisConfigPath,
			params.genesisKeyOut,
		)
		s.Serve()
		return
	}

	var err error
	cfgHolder := configuration.NewHolder()
	if len(params.configPath) != 0 {
		err = cfgHolder.LoadFromFile(params.configPath)
	} else {
		err = cfgHolder.Load()
	}
	if err != nil {
		log.Error("failed to load configuration from file: ", err.Error())
		return
	}
	data, err := ioutil.ReadFile(filepath.Clean(cfgHolder.Configuration.CertificatePath))
	if err != nil {
		log.Error(errors.Wrapf(err, "[ ReadCertificate ] failed to read certificate from: %s", cfgHolder.Configuration.CertificatePath))
		return
	}
	cert := certificate.AuthorizationCertificate{}
	err = json.Unmarshal(data, &cert)
	if err != nil {
		log.Error(errors.Wrap(err, "[ newCertificate ] failed to parse certificate json"))
		return
	}
	switch cert.GetRole() {
	case insolar.StaticRoleHeavyMaterial, insolar.StaticRoleLightMaterial:
		s := server.NewLightServer(params.configPath, params.traceEnabled)
		s.Serve()
	case insolar.StaticRoleVirtual:
		s := server.NewVirtualServer(params.configPath, params.traceEnabled)
		s.Serve()
	}
}
