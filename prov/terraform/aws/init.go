package aws

import "github.com/ystia/yorc/registry"
import "github.com/ystia/yorc/prov/terraform"

func init() {
	reg := registry.GetRegistry()
	reg.RegisterDelegates([]string{`yorc\.nodes\.aws\..*`}, terraform.NewExecutor(&awsGenerator{}, nil), registry.BuiltinOrigin)
}
