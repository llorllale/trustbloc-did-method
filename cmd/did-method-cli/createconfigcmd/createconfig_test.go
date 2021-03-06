/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package createconfigcmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/cobra"

	docdid "github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/trustbloc/trustbloc-did-method/pkg/did"

	"github.com/stretchr/testify/require"
)

const flag = "--"

// nolint: gochecknoglobals
var configData = `{
  "consortium_data": {
    "domain": "consortium.net",
    "policy": {
      "cache": {
        "max_age": 2419200
      },
      "num_queries": 2,
      "history_hash": "SHA256",
      "sidetree": {
        "hash_algorithm": "SHA256",
        "key_algorithm": "NotARealAlg2018",
        "max_encoded_hash_length": 100,
        "max_operation_size": 8192
      }
    }
  },
  "members_data": [
    {
      "domain": "stakeholder.one",
      "policy": {"cache": {"max_age": 604800}},
      "endpoints": [
        "http://endpoints.stakeholder.one/peer1/",
        "http://endpoints.stakeholder.one/peer2/"
      ],
      "privateKeyJwkPath": "%s"
    }
  ]
}`

// nolint: gochecknoglobals
var jwkData = `{
        "kty": "OKP",
        "kid": "key1",
        "d": "-YawjZSeB9Rkdol9SHeOcT9hIvo_VuH6zM-pgtk3b10",
        "crv": "Ed25519",
        "x": "bWRCy8DtNhRO3HdKTFB2eEG5Ac1J00D0DQPffOwtAD0"
      }`

func TestCreateConfigCmdWithMissingArg(t *testing.T) {
	t.Run("test missing arg sidetree url", func(t *testing.T) {
		cmd := GetCreateConfigCmd()

		err := cmd.Execute()
		require.Error(t, err)
		require.Equal(t,
			"Neither sidetree-url (command line flag) nor DID_METHOD_CLI_SIDETREE_URL (environment variable) have been set.",
			err.Error())
	})

	t.Run("test missing arg config file", func(t *testing.T) {
		cmd := GetCreateConfigCmd()

		cmd.SetArgs(sidetreeURLArg())
		err := cmd.Execute()

		require.Error(t, err)
		require.Equal(t,
			"Neither config-file (command line flag) nor DID_METHOD_CLI_CONFIG_FILE (environment variable) have been set.",
			err.Error())
	})
}

func TestCreateConfigCmd(t *testing.T) {
	t.Run("test wrong config file", func(t *testing.T) {
		cmd := GetCreateConfigCmd()

		var args []string
		args = append(args, sidetreeURLArg()...)
		args = append(args, configFileArg("wrongValue")...)

		cmd.SetArgs(args)

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("test wrong path for private key jwk", func(t *testing.T) {
		cmd := GetCreateConfigCmd()

		file, err := ioutil.TempFile("", "*.json")
		require.NoError(t, err)

		_, err = file.WriteString(fmt.Sprintf(configData, "notexist.json"))
		require.NoError(t, err)

		defer func() { require.NoError(t, os.Remove(file.Name())) }()

		var args []string
		args = append(args, sidetreeURLArg()...)
		args = append(args, configFileArg(file.Name())...)

		cmd.SetArgs(args)

		err = cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read jwk file")
	})

	t.Run("test wrong private key jwk", func(t *testing.T) {
		cmd := GetCreateConfigCmd()

		jwkFile, err := ioutil.TempFile("", "*.json")
		require.NoError(t, err)

		defer func() { require.NoError(t, os.Remove(jwkFile.Name())) }()

		_, err = jwkFile.WriteString("wrongjwk")
		require.NoError(t, err)

		file, err := ioutil.TempFile("", "*.json")
		require.NoError(t, err)

		_, err = file.WriteString(fmt.Sprintf(configData, jwkFile.Name()))
		require.NoError(t, err)

		defer func() { require.NoError(t, os.Remove(file.Name())) }()

		var args []string
		args = append(args, sidetreeURLArg()...)
		args = append(args, configFileArg(file.Name())...)

		cmd.SetArgs(args)

		err = cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal to jwk")
	})

	t.Run("test error from create did", func(t *testing.T) {
		cmd := GetCreateConfigCmd()

		jwkFile, err := ioutil.TempFile("", "*.json")
		require.NoError(t, err)

		defer func() { require.NoError(t, os.Remove(jwkFile.Name())) }()

		_, err = jwkFile.WriteString(jwkData)
		require.NoError(t, err)

		file, err := ioutil.TempFile("", "*.json")
		require.NoError(t, err)

		_, err = file.WriteString(fmt.Sprintf(configData, jwkFile.Name()))
		require.NoError(t, err)

		defer func() { require.NoError(t, os.Remove(file.Name())) }()

		var args []string
		args = append(args, sidetreeURLArg()...)
		args = append(args, configFileArg(file.Name())...)

		cmd.SetArgs(args)

		err = cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to send create sidetree request")
	})

	t.Run("test create config and write them to file", func(t *testing.T) {
		os.Clearenv()

		jwkFile, err := ioutil.TempFile("", "*.json")
		require.NoError(t, err)

		defer func() { require.NoError(t, os.Remove(jwkFile.Name())) }()

		_, err = jwkFile.WriteString(jwkData)
		require.NoError(t, err)

		file, err := ioutil.TempFile("", "*.json")
		require.NoError(t, err)

		_, err = file.WriteString(fmt.Sprintf(configData, jwkFile.Name()))
		require.NoError(t, err)

		defer func() { require.NoError(t, os.Remove(file.Name())) }()

		require.NoError(t, os.Setenv(configFileEnvKey, file.Name()))

		c, err := getConfig(&cobra.Command{})
		require.NoError(t, err)

		filesData, err := createConfig(&parameters{config: c,
			didClient: &mockDIDClient{&docdid.Doc{ID: "did:test:123"}}})
		require.NoError(t, err)

		require.Equal(t, 2, len(filesData))

		dir, err := ioutil.TempDir("", "")
		require.NoError(t, err)

		defer func() { require.NoError(t, os.RemoveAll(dir)) }()

		require.NoError(t, writeConfig(dir, filesData))

		_, err = os.Stat(dir + "/consortium.net.json")
		require.False(t, os.IsNotExist(err))

		_, err = os.Stat(dir + "/stakeholder.one.json")
		require.False(t, os.IsNotExist(err))
	})
}

func TestTLSSystemCertPoolInvalidArgsEnvVar(t *testing.T) {
	os.Clearenv()

	startCmd := GetCreateConfigCmd()

	require.NoError(t, os.Setenv(sidetreeURLEnvKey, "localhost:8080"))
	require.NoError(t, os.Setenv(configFileEnvKey, "domain"))
	require.NoError(t, os.Setenv(tlsSystemCertPoolEnvKey, "wrongvalue"))

	err := startCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid syntax")
}

func sidetreeURLArg() []string {
	return []string{flag + sidetreeURLFlagName, "localhost:8080"}
}

func configFileArg(config string) []string {
	return []string{flag + configFileFlagName, config}
}

type mockDIDClient struct {
	createDIDValue *docdid.Doc
}

func (m *mockDIDClient) CreateDID(domain string, opts ...did.CreateDIDOption) (*docdid.Doc, error) {
	return m.createDIDValue, nil
}
