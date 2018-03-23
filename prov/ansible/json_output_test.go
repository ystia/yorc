// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ansible

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/ystia/yorc/log"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/stringutil"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/testutil"
)

func testLogAnsibleOutputInConsul(t *testing.T, kv *api.KV) {
	data := `
{
    "plays": [
        {
            "play": {
                "id": "2363d576-d513-407d-9539-75fce3c71523",
                "name": "Connects consul agent to consul server"
            },
            "tasks": [
                {
                    "hosts": {
                        "10.197.138.212": {
                            "_ansible_no_log": false,
                            "_ansible_parsed": true,
                            "_ansible_verbose_override": true,
                            "ansible_facts": {
                            },
                            "changed": false,
                            "invocation": {
                                "module_args": {
                                    "fact_path": "/etc/ansible/facts.d",
                                    "filter": "*",
                                    "gather_subset": [
                                        "all"
                                    ],
                                    "gather_timeout": 10
                                },
                                "module_name": "setup"
                            }
                        }
                    },
                    "task": {
                        "id": "66de15cb-54d2-4f94-88e8-1485e1bd0887",
                        "name": ""
                    }
                },
                {
                    "hosts": {
                        "10.197.138.212": {
                            "_ansible_no_log": false,
                            "_ansible_parsed": true,
                            "changed": true,
                            "dest": "/home/cloud-user/consul/work/.agentmode",
                            "diff": {
                                "after": {
                                    "path": "/home/cloud-user/consul/work/.agentmode",
                                    "state": "touch"
                                },
                                "before": {
                                    "path": "/home/cloud-user/consul/work/.agentmode",
                                    "state": "file"
                                }
                            },
                            "gid": 1000,
                            "group": "cloud-user",
                            "invocation": {
                                "module_args": {
                                    "backup": null,
                                    "content": null,
                                    "delimiter": null,
                                    "dest": "~/consul/work/.agentmode",
                                    "diff_peek": null,
                                    "directory_mode": null,
                                    "follow": false,
                                    "force": false,
                                    "group": null,
                                    "mode": null,
                                    "original_basename": null,
                                    "owner": null,
                                    "path": "/home/cloud-user/consul/work/.agentmode",
                                    "recurse": false,
                                    "regexp": null,
                                    "regexp": null,
                                    "remote_src": null,
                                    "selevel": null,
                                    "serole": null,
                                    "setype": null,
                                    "seuser": null,
                                    "src": null,
                                    "state": "touch",
                                    "unsafe_writes": null,
                                    "validate": null
                                },
                                "module_name": "file"
                            },
                            "mode": "0664",
                            "owner": "cloud-user",
                            "secontext": "unconfined_u:object_r:user_home_t:s0",
                            "size": 0,
                            "state": "file",
                            "uid": 1000
                        }
                    },
                    "task": {
                        "id": "4c44a1c1-ca1f-4bc8-9f90-6aab51ef01fc",
                        "name": "set agent flag"
                    }
                },
                {
                    "hosts": {
                        "10.197.138.212": {
                            "changed": false,
                            "msg": "All items completed",
                            "results": [
                                {
                                    "_ansible_item_result": true,
                                    "_ansible_no_log": false,
                                    "ansible_facts": {
                                        "consul_servers": [
                                            "10.0.0.176"
                                        ]
                                    },
                                    "changed": false,
                                    "invocation": {
                                        "module_args": {
                                            "consul_servers": [
                                                "10.0.0.176"
                                            ]
                                        },
                                        "module_name": "set_fact"
                                    },
                                    "item": "ConsulServer_0"
                                }
                            ]
                        }
                    },
                    "task": {
                        "id": "b3f418f1-4560-49a6-bb2e-1efa2b0f7843",
                        "name": "compute consul servers"
                    }
                },
                {
                    "hosts": {
                        "10.197.138.212": {
                            "_ansible_no_log": false,
                            "_ansible_parsed": true,
                            "changed": false,
                            "diff": {
                                "after": {
                                    "path": "/home/cloud-user/consul/config/2_connects_to_servers.json"
                                },
                                "before": {
                                    "path": "/home/cloud-user/consul/config/2_connects_to_servers.json"
                                }
                            },
                            "gid": 1000,
                            "group": "cloud-user",
                            "invocation": {
                                "module_args": {
                                    "backup": null,
                                    "content": null,
                                    "delimiter": null,
                                    "dest": "~/consul/config/2_connects_to_servers.json",
                                    "diff_peek": null,
                                    "directory_mode": null,
                                    "follow": true,
                                    "force": false,
                                    "group": null,
                                    "mode": null,
                                    "original_basename": "2_connects_to_servers.json.j2",
                                    "owner": null,
                                    "path": "/home/cloud-user/consul/config/2_connects_to_servers.json",
                                    "recurse": false,
                                    "regexp": null,
                                    "remote_src": null,
                                    "selevel": null,
                                    "serole": null,
                                    "setype": null,
                                    "seuser": null,
                                    "src": null,
                                    "state": null,
                                    "unsafe_writes": null,
                                    "validate": null
                                }
                            },
                            "mode": "0664",
                            "owner": "cloud-user",
                            "path": "/home/cloud-user/consul/config/2_connects_to_servers.json",
                            "secontext": "unconfined_u:object_r:user_home_t:s0",
                            "size": 70,
                            "state": "file",
                            "uid": 1000
                        }
                    },
                    "task": {
                        "id": "acdd72a8-18b6-4721-ae6e-e82c3fb46e39",
                        "name": "Install servers config for consul"
                    }
                },
                {
                    "hosts": {
                        "10.197.138.212": {
                            "_ansible_no_log": false,
                            "_ansible_verbose_always": true,
                            "changed": false,
                            "invocation": {
                                "module_args": {
                                    "msg": "Consul Agent configured to connects to server on [10.0.0.176]"
                                },
                                "module_name": "debug"
                            },
                            "msg": "Consul Agent configured to connects to server on [10.0.0.176]"
                        }
                    },
                    "task": {
                        "id": "2226e1f8-ce7d-4423-adae-7490697d1e7e",
                        "name": "echo servers list"
                    }
                }
            ]
        }
    ],
    "stats": {
        "10.197.138.212": {
            "changed": 1,
            "failures": 0,
            "ok": 5,
            "skipped": 0,
            "unreachable": 0
        }
    }
}

`

	log.SetDebug(true)
	deploymentID := testutil.BuildDeploymentID(t)
	ec := &executionCommon{kv: kv, deploymentID: deploymentID, NodeName: "node"}
	ea := &executionAnsible{executionCommon: ec}
	var buf bytes.Buffer
	buf.WriteString(data)

	// Fill log optional fields for log registration
	wfName, _ := tasks.GetTaskData(ec.kv, ec.taskID, "workflowName")
	logOptFields := events.LogOptionalFields{
		events.WorkFlowID:    wfName,
		events.NodeID:        ec.NodeName,
		events.OperationName: stringutil.GetLastElement(ec.operation.Name, "."),
		events.InterfaceName: stringutil.GetAllExceptLastElement(ec.operation.Name, "."),
	}
	ctx := events.NewContext(context.Background(), logOptFields)
	err := ea.logAnsibleOutputInConsul(ctx, &buf)
	t.Logf("%+v", err)
	require.Nil(t, err)

	logs, _, err := events.LogsEvents(kv, deploymentID, 0, 5*time.Millisecond)
	require.Nil(t, err)
	require.Len(t, logs, 1)

}
