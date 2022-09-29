package envconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_variable_loadFromFile(t *testing.T) {
	type data struct {
		secretValue string
		setEnvKey   string
		fileSuffix  string
		opts        []Option
	}

	tests := []struct {
		name     string
		data     data
		expected string
	}{
		{
			data: data{
				secretValue: "qwerty",
				setEnvKey:   "ENV_CONFIG_SECRET" + DefaultFileSuffix,
				fileSuffix:  "",
				opts: []Option{
					WithPrefix("env_config"),
				},
			},
			expected: "qwerty",
		},
		{
			data: data{
				secretValue: "qwerty",
				setEnvKey:   "ENV_CONFIG_SECRET_FOO",
				fileSuffix:  "_FOO",
				opts: []Option{
					WithPrefix("env_config"),
					WithDefaultFileSuffix("_FOO"),
				},
			},
			expected: "qwerty",
		},
		{
			data: data{
				secretValue: "qwerty",
				setEnvKey:   "ENV_CONFIG_SECRET_FOO",
				fileSuffix:  "_FOO",
				opts: []Option{
					WithPrefix("env_config"),
					WithDefaultFileSuffix("_FOO"),
					WithoutDefaultLoadingFromFiles(),
				},
			},
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretFile, err := os.CreateTemp("", "envconfig_test_secret")
			if err != nil {
				t.Error(err)
			}
			defer func() {
				removeErr := os.Remove(secretFile.Name())
				if removeErr != nil {
					t.Error(removeErr)
				}
			}()

			_, err = secretFile.WriteString(tt.data.secretValue)
			if err != nil {
				t.Error(err)
			}

			var s struct {
				Secret string
			}

			os.Clearenv()
			os.Setenv(tt.data.setEnvKey, secretFile.Name())

			err = Process(&s, tt.data.opts...)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, s.Secret)
		})
	}
}

func Test_variable_loadFromFile_enabledByTag(t *testing.T) {
	type data struct {
		secretValue string
		setEnvKey   string
		fileSuffix  string
		opts        []Option
	}

	tests := []struct {
		name     string
		data     data
		expected string
	}{
		{
			data: data{
				secretValue: "qwerty",
				setEnvKey:   "ENV_CONFIG_SECRET" + DefaultFileSuffix,
				fileSuffix:  "",
				opts: []Option{
					WithPrefix("env_config"),
					WithoutDefaultLoadingFromFiles(),
				},
			},
			expected: "",
		},
		{
			data: data{
				secretValue: "qwerty",
				setEnvKey:   "ENV_CONFIG_SECRET" + DefaultFileSuffix,
				fileSuffix:  "",
				opts: []Option{
					WithPrefix("env_config"),
				},
			},
			expected: "",
		},
		{
			data: data{
				secretValue: "qwerty",
				setEnvKey:   "ENV_CONFIG_SECRET_FOO",
				fileSuffix:  "_FOO",
				opts: []Option{
					WithPrefix("env_config"),
					WithDefaultFileSuffix("_FOO"),
				},
			},
			expected: "qwerty",
		},
		{
			data: data{
				secretValue: "qwerty",
				setEnvKey:   "ENV_CONFIG_SECRET_FOO",
				fileSuffix:  "_FOO",
				opts: []Option{
					WithPrefix("env_config"),
					WithoutDefaultLoadingFromFiles(),
				},
			},
			expected: "qwerty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretFile, err := os.CreateTemp("", "envconfig_test_secret")
			if err != nil {
				t.Error(err)
			}
			defer func() {
				removeErr := os.Remove(secretFile.Name())
				if removeErr != nil {
					t.Error(removeErr)
				}
			}()

			_, err = secretFile.WriteString(tt.data.secretValue)
			if err != nil {
				t.Error(err)
			}

			var s struct {
				Secret string `file:"_FOO"`
			}

			os.Clearenv()
			os.Setenv(tt.data.setEnvKey, secretFile.Name())

			err = Process(&s, tt.data.opts...)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, s.Secret)
		})
	}
}
