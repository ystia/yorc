package tosca

import "github.com/ystia/yorc/registry"
import "github.com/ystia/yorc/log"

func init() {
	reg := registry.GetRegistry()
	for _, defName := range AssetNames() {
		definition, err := Asset(defName)
		if err != nil {
			log.Panicf("Failed to load builtin Tosca definition file %q. %v", defName, err)
		}
		reg.AddToscaDefinition(defName, registry.BuiltinOrigin, definition)
	}
}
