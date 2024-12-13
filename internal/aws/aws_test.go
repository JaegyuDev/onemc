package aws

import "testing"

func TestCheckAWSInstanceStatus(t *testing.T) {
	IsAWSInstanceRunning("i-0d72f9a87b77147e0")
}
