package training

import (
	"testing"
	"time"
)

func TestNewTrainingConfig(t *testing.T) {
	// Test inputs
	testBucket := "my-test-bucket"
	testOutputPath := "s3://my-test-bucket/model-output/"
	
	// Create config using the fixed function
	config := NewTrainingConfig(testBucket, testOutputPath)
	
	// Verify all required fields are initialized (fixing the bug)
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"TrainingDataBucket", config.TrainingDataBucket, testBucket},
		{"OutputPath", config.OutputPath, testOutputPath},
		{"TrainingInstanceType", config.TrainingInstanceType, "ml.m5.large"},
		{"TrainingImage", config.TrainingImage, "382416733822.dkr.ecr.us-east-1.amazonaws.com/xgboost:latest"},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.value != test.expected {
				t.Errorf("%s: got %v, expected %v", test.name, test.value, test.expected)
			}
		})
	}
	
	// Verify numeric fields are not zero (bug fix)
	if config.TrainingInstanceCount == 0 {
		t.Errorf("TrainingInstanceCount should not be zero (bug fix)")
	}
	
	if config.VolumeSize == 0 {
		t.Errorf("VolumeSize should not be zero (bug fix)")
	}
	
	if config.MaxRuntimeInSeconds == 0 {
		t.Errorf("MaxRuntimeInSeconds should not be zero (bug fix)")
	}
	
	// Verify string fields are not empty (bug fix)
	if config.ModelName == "" {
		t.Errorf("ModelName should not be empty (bug fix)")
	}
	
	if config.TrainingJobName == "" {
		t.Errorf("TrainingJobName should not be empty (bug fix)")
	}
	
	if config.RoleArn == "" {
		t.Errorf("RoleArn should not be empty (bug fix)")
	}
	
	if config.TrainingImage == "" {
		t.Errorf("TrainingImage should not be empty (bug fix)")
	}
	
	if config.TrainingInstanceType == "" {
		t.Errorf("TrainingInstanceType should not be empty (bug fix)")
	}
	
	// Verify HyperParameters map is not nil (bug fix)
	if config.HyperParameters == nil {
		t.Errorf("HyperParameters should not be nil (bug fix)")
	}
	
	// Verify HyperParameters has expected keys
	expectedHyperParams := []string{"objective", "num_round", "max_depth", "eta", "subsample", "colsample_bytree"}
	for _, key := range expectedHyperParams {
		if _, exists := config.HyperParameters[key]; !exists {
			t.Errorf("HyperParameters missing key: %s", key)
		}
	}
	
	// Verify timestamp-based fields are unique
	config2 := NewTrainingConfig(testBucket, testOutputPath)
	time.Sleep(1 * time.Second) // Ensure different timestamp
	config3 := NewTrainingConfig(testBucket, testOutputPath)
	
	if config2.ModelName == config3.ModelName {
		t.Errorf("ModelName should be unique between calls")
	}
	
	if config2.TrainingJobName == config3.TrainingJobName {
		t.Errorf("TrainingJobName should be unique between calls")
	}
}

func TestNewTrainingConfigWithDefaults(t *testing.T) {
	// Test with partial parameters
	params := TrainingConfigParams{
		TrainingDataBucket: "custom-bucket",
		OutputPath:         "s3://custom-bucket/output/",
		TrainingInstanceType: "ml.m5.xlarge",
		// Other fields intentionally left empty to test defaults
	}
	
	config := NewTrainingConfigWithDefaults(params)
	
	// Verify custom values are preserved
	if config.TrainingDataBucket != "custom-bucket" {
		t.Errorf("TrainingDataBucket: got %s, expected custom-bucket", config.TrainingDataBucket)
	}
	
	if config.TrainingInstanceType != "ml.m5.xlarge" {
		t.Errorf("TrainingInstanceType: got %s, expected ml.m5.xlarge", config.TrainingInstanceType)
	}
	
	// Verify defaults are applied for empty fields
	if config.ModelName == "" {
		t.Errorf("ModelName should have default value")
	}
	
	if config.TrainingJobName == "" {
		t.Errorf("TrainingJobName should have default value")
	}
	
	if config.RoleArn == "" {
		t.Errorf("RoleArn should have default value")
	}
	
	if config.TrainingImage == "" {
		t.Errorf("TrainingImage should have default value")
	}
	
	if config.TrainingInstanceCount == 0 {
		t.Errorf("TrainingInstanceCount should have default value")
	}
	
	if config.VolumeSize == 0 {
		t.Errorf("VolumeSize should have default value")
	}
	
	if config.MaxRuntimeInSeconds == 0 {
		t.Errorf("MaxRuntimeInSeconds should have default value")
	}
	
	if config.HyperParameters == nil {
		t.Errorf("HyperParameters should have default value")
	}
}

// TestBugFix demonstrates the specific bug that was fixed
func TestBugFix(t *testing.T) {
	config := NewTrainingConfig("test-bucket", "s3://test-bucket/output/")
	
	// Before the fix, these fields would be empty/zero, causing CreateTrainingJob to fail
	requiredFields := map[string]interface{}{
		"ModelName":              config.ModelName,
		"TrainingJobName":        config.TrainingJobName,
		"RoleArn":               config.RoleArn,
		"TrainingImage":         config.TrainingImage,
		"TrainingInstanceType":  config.TrainingInstanceType,
		"TrainingInstanceCount": config.TrainingInstanceCount,
		"VolumeSize":           config.VolumeSize,
		"MaxRuntimeInSeconds":  config.MaxRuntimeInSeconds,
		"HyperParameters":      config.HyperParameters,
	}
	
	for fieldName, value := range requiredFields {
		switch v := value.(type) {
		case string:
			if v == "" {
				t.Errorf("BUG: %s is empty, would cause CreateTrainingJob to fail", fieldName)
			}
		case int:
			if v == 0 {
				t.Errorf("BUG: %s is zero, would cause CreateTrainingJob to fail", fieldName)
			}
		case map[string]string:
			if v == nil {
				t.Errorf("BUG: %s is nil, would cause CreateTrainingJob to fail", fieldName)
			}
		}
	}
}