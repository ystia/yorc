package ansible

import (
	"bytes"
	"fmt"
	"github.com/antonholmquist/jason"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
	"time"
)

func (e *execution) logAnsibleOutputInConsul(output *bytes.Buffer) error {
	// Workaround https://github.com/ansible/ansible/issues/17122
	b := output.Bytes()
	if i := bytes.Index(b, []byte("{")); i >= 0 {
		b = b[i:]
	} else {
		return errors.New("Not a valid JSON output")
	}
	//Construct the JSON from the buffer
	v, err := jason.NewObjectFromBytes(b)
	if err != nil {
		err = errors.Wrap(err, "Ansible logs not available")
		log.Printf("%v", err)
		log.Debugf("%+v", err)
		log.Debugf("String: %q", string(b))
		return err
	}

	//Get the array of object of plays
	plays, err := v.GetObjectArray("plays")
	for _, play := range plays {
		//Extract the tasks from the play
		tasks, err := play.GetObjectArray("tasks")
		if err != nil {
			err = errors.Wrap(err, "Ansible logs not available")
			log.Printf("%v", err)
			log.Debugf("%+v", err)
			continue
		}
		for _, task := range tasks {
			//Extract the hosts object from the  tasks
			tmp, err := task.GetObject("hosts")
			if err != nil {
				err = errors.Wrap(err, "Ansible logs not available")
				log.Printf("%v", err)
				log.Debugf("%+v", err)
				continue
			}
			//Convert the host into map like ["IP_ADDR"]Json_Object
			mapTmp := tmp.Map()
			//Iterate on this map (normally a single object)
			for host, v := range mapTmp {
				//Convert the value in Object type
				obj, err := v.Object()
				if err != nil {
					err = errors.Wrap(err, "Ansible logs not available")
					log.Printf("%v", err)
					log.Debugf("%+v", err)
					continue
				}
				//Check if a stderr field is present (The stdout field is exported for shell tasks on ansible)
				if std, err := obj.GetString("stderr"); err == nil && std != "" {
					//Display it and store it in consul
					log.Debugf("Stderr found on host : %s  message : %s", host, std)
					key := path.Join(consulutil.DeploymentKVPrefix, e.DeploymentId, "logs", deployments.SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano))
					err = consulutil.StoreConsulKeyAsString(key, fmt.Sprintf("node %q, host %q, stderr:\n%s", e.NodeName, host, std))
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
				}
				//Check if a stdout field is present (The stdout field is exported for shell tasks on ansible)
				if std, err := obj.GetString("stdout"); err == nil && std != "" {
					//Display it and store it in consul
					log.Debugf("Stdout found on host : %s  message : %s", host, std)
					key := path.Join(consulutil.DeploymentKVPrefix, e.DeploymentId, "logs", deployments.SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano))
					err = consulutil.StoreConsulKeyAsString(key, fmt.Sprintf("node %q, host %q, stdout:\n%s", e.NodeName, host, std))
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
				}

				//Check if a msg field is present (The stdout field is exported for shell tasks on ansible)
				if std, err := obj.GetString("msg"); err == nil && std != "" {
					//Display it and store it in consul
					log.Debugf("Stdout found on host : %s  message : %s", host, std)
					key := path.Join(consulutil.DeploymentKVPrefix, e.DeploymentId, "logs", deployments.SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano))
					err = consulutil.StoreConsulKeyAsString(key, fmt.Sprintf("node %q, host %q, msg:\n%s", e.NodeName, host, std))
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
				}
			}
		}

	}

	return nil
}
