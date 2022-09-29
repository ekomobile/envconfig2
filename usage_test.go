package envconfig

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/stretchr/testify/assert"
)

var testUsageTableResult, testUsageListResult, testUsageCustomResult, testUsageBadFormatResult string

func TestMain(m *testing.M) {

	// Load the expected test results from a text file
	data, err := ioutil.ReadFile("testdata/default_table.txt")
	if err != nil {
		log.Fatal(err)
	}
	testUsageTableResult = string(data)

	data, err = ioutil.ReadFile("testdata/default_list.txt")
	if err != nil {
		log.Fatal(err)
	}
	testUsageListResult = string(data)

	data, err = ioutil.ReadFile("testdata/custom.txt")
	if err != nil {
		log.Fatal(err)
	}
	testUsageCustomResult = string(data)

	data, err = ioutil.ReadFile("testdata/fault.txt")
	if err != nil {
		log.Fatal(err)
	}
	testUsageBadFormatResult = string(data)

	retCode := m.Run()
	os.Exit(retCode)
}

func compareUsage(want, got string, t *testing.T) {
	got = strings.ReplaceAll(got, " ", ".")
	if want != got {
		shortest := len(want)
		if len(got) < shortest {
			shortest = len(got)
		}
		if len(want) != len(got) {
			t.Errorf("expected result length of %d, found %d", len(want), len(got))
		}
		for i := 0; i < shortest; i++ {
			if want[i] != got[i] {
				t.Errorf("difference at index %d, expected '%c' (%v), found '%c' (%v)\n",
					i, want[i], want[i], got[i], got[i])
				break
			}
		}
		t.Errorf("Complete Expected:\n'%s'\nComplete Found:\n'%s'\n", want, got)
	}
}

func TestUsageDefault(t *testing.T) {
	var s Specification
	os.Clearenv()
	save := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := Usage(&s, WithPrefix("env_config"))
	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()
	w.Close()
	os.Stdout = save // restoring the real stdout
	out := <-outC

	if err != nil {
		t.Error(err.Error())
	}
	compareUsage(testUsageTableResult, out, t)
}

func TestUsageTable(t *testing.T) {
	var s Specification
	os.Clearenv()
	buf := new(bytes.Buffer)
	tabs := tabwriter.NewWriter(buf, 1, 0, 4, ' ', 0)
	err := Usagef(&s, tabs, DefaultTableFormat, WithPrefix("env_config"))
	tabs.Flush()
	if err != nil {
		t.Error(err.Error())
	}
	compareUsage(testUsageTableResult, buf.String(), t)
}

func TestUsageList(t *testing.T) {
	var s Specification
	os.Clearenv()
	buf := new(bytes.Buffer)
	err := Usagef(&s, buf, DefaultListFormat, WithPrefix("env_config"))
	if err != nil {
		t.Error(err.Error())
	}
	compareUsage(testUsageListResult, buf.String(), t)
}

func TestUsageCustomFormat(t *testing.T) {
	var s Specification
	os.Clearenv()
	buf := new(bytes.Buffer)
	err := Usagef(&s, buf, "{{range .}}{{usage_key .}}={{usage_description .}}\n{{end}}", WithPrefix("env_config"))
	if err != nil {
		t.Error(err.Error())
	}
	compareUsage(testUsageCustomResult, buf.String(), t)
}

func TestUsageUnknownKeyFormat(t *testing.T) {
	var s Specification
	unknownError := "template: envconfig:1:2: executing \"envconfig\" at <.UnknownKey>"
	os.Clearenv()
	buf := new(bytes.Buffer)
	err := Usagef(&s, buf, "{{.UnknownKey}}", WithPrefix("env_config"))
	if assert.Errorf(t, err, "expected 'unknown key' error, but got no error") {
		assert.Contains(t, err.Error(), unknownError)
	}
}

func TestUsageBadFormat(t *testing.T) {
	var s Specification
	os.Clearenv()
	// If you don't use two {{}} then you get a lieteral
	buf := new(bytes.Buffer)
	err := Usagef(&s, buf, "{{range .}}{.key}\n{{end}}", WithPrefix("env_config"))
	assert.NoError(t, err)

	compareUsage(testUsageBadFormatResult, buf.String(), t)
}
