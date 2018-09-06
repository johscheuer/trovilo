package job

import (
	"fmt"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	actualJob, err := GetJob(nil, "../examples/local/trovilo-config.yaml")

	if err != nil {
		t.FailNow()
	}

	expectedJob := Job{
		Name: "alert-rules",
		Selector: map[string]string{
			"type": "prometheus-alerts",
		},
		Verify: []VerifyStep{
			VerifyStep{
				Name: "verify alert rule validity",
				Cmd: VerifyStepCmd{
					"promtool",
					"check",
					"rules",
					"%s",
				},
			},
		},
		TargetDir: "/etc/prometheus-alerts/",
		Flatten:   true,
		PostDeploy: []PostDeployAction{
			PostDeployAction{
				Name: "reload prometheus",
				Cmd: PostDeployActionCmd{
					"curl",
					"-s",
					"-X",
					"POST",
					"http://localhost:9090/-/reload",
				},
			},
		},
	}

	if !reflect.DeepEqual(actualJob, expectedJob) {
		fmt.Printf("Expected: %v\nGot: %v", expectedJob, actualJob)
		t.FailNow()
	}
}

// TODO more tests!
