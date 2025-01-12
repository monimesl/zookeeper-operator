/*
 * Copyright 2021 - now, the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package zookeepercluster

import (
	"context"
	"fmt"
	"github.com/monimesl/operator-helper/oputil"
	"github.com/monimesl/operator-helper/reconciler"
	"github.com/monimesl/zookeeper-operator/api/v1alpha1"
	"github.com/monimesl/zookeeper-operator/internal/zk"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"strconv"
)

// ReconcileConfigMap reconcile the configmap of the specified cluster
func ReconcileConfigMap(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	cm := &v1.ConfigMap{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.GetName(),
		Namespace: cluster.Namespace,
	}, cm,
		// Found
		func() error {
			if shouldUpdateConfigmap(ctx, cluster) {
				if err := updateConfigmap(ctx, cm, cluster); err != nil {
					return err
				}
			}
			return nil
		},
		// Not Found
		func() (err error) {
			cm = createConfigMap(cluster)
			if err = ctx.SetOwnershipReference(cluster, cm); err == nil {
				ctx.Logger().Info("Creating the zookeeper configMap",
					"ConfigMap.Name", cm.GetName(),
					"ConfigMap.Namespace", cm.GetNamespace())
				if err = ctx.Client().Create(context.TODO(), cm); err == nil {
					ctx.Logger().Info("ConfigMap creation success.",
						"ConfigMap.Name", cm.GetName(),
						"ConfigMap.Namespace", cm.GetNamespace())
				}
			}
			return
		})
}

func createConfigMap(c *v1alpha1.ZookeeperCluster) *v1.ConfigMap {
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.GetName(),
			Labels:    c.GenerateLabels(),
			Namespace: c.GetNamespace(),
		},
		Data: createConfigmapData(c),
	}
}

func shouldUpdateConfigmap(ctx reconciler.Context, c *v1alpha1.ZookeeperCluster) bool {
	if c.Spec.ZkConfig != c.Status.Metadata.ZkConfig {
		ctx.Logger().Info("Zookeeper cluster config changed",
			"from", c.Spec.ZkConfig, "to", c.Status.Metadata.ZkConfig)
		return true
	}
	return false
}

func updateConfigmap(ctx reconciler.Context, cm *v1.ConfigMap, c *v1alpha1.ZookeeperCluster) error {
	ctx.Logger().Info("Updating the zookeeper configmap.",
		"configMap.Name", cm.GetName(),
		"ConfigMap.Namespace", cm.GetNamespace())
	cm.Labels = c.GenerateLabels()
	cm.Data = createConfigmapData(c)
	return ctx.Client().Update(context.TODO(), cm)
}

func createConfigmapData(c *v1alpha1.ZookeeperCluster) map[string]string {
	return map[string]string{
		"zoo.cfg":     createZkConfig(c),
		"bootEnv.sh":  createBootEnvScript(c),
		"logback.xml": createZkLogbackXmData(c),
	}
}

func createBootEnvScript(c *v1alpha1.ZookeeperCluster) string {
	return "#!/usr/bin/env bash\n\n" +
		fmt.Sprintf("CLUSTER_NAME=%s\n", c.GetName()) +
		fmt.Sprintf("CLUSTER_METADATA_PARENT_ZNODE=%s\n", zk.ClusterMetadataParentZNode) +
		fmt.Sprintf("DATA_DIR=%s\n", c.Spec.Directories.Data) +
		fmt.Sprintf("CLIENT_PORT=%d\n", c.Spec.Ports.Client) +
		fmt.Sprintf("SECURE_CLIENT_PORT=%d\n", c.Spec.Ports.SecureClient) +
		fmt.Sprintf("QUORUM_PORT=%d\n", c.Spec.Ports.Quorum) +
		fmt.Sprintf("LEADER_PORT=%d\n", c.Spec.Ports.Leader)
}

func createZkConfig(c *v1alpha1.ZookeeperCluster) string {
	clientPort := fmt.Sprintf("%d", c.Spec.Ports.Client)
	metricsPort := fmt.Sprintf("%d", c.Spec.Ports.Metrics)
	secureClientPort := fmt.Sprintf("%d", c.Spec.Ports.SecureClient)
	if c.Spec.Ports.Client <= 0 {
		clientPort = ""
	}
	if !c.IsSslClientSupported() {
		secureClientPort = ""
	}
	enableAdmin := c.Spec.Ports.Admin > 0
	str, _ := oputil.CreateConfigFromYamlString(c.Spec.ZkConfig, "zoo.cfg", map[string]string{
		"initLimit":              "10",
		"syncLimit":              "5",
		"tickTime":               "2000",
		"skipACL":                "yes",
		"reconfigEnabled":        "true",
		"standaloneEnabled":      "false",
		"clientPort":             clientPort,
		"secureClientPort":       secureClientPort,
		"dataDir":                c.Spec.Directories.Data,
		"dataLogDir":             c.Spec.Directories.Log,
		"dynamicConfigFile":      fmt.Sprintf("%s/conf/zoo.cfg.dynamic", c.Spec.Directories.Data),
		"4lw.commands.whitelist": "conf, cons, crst, conf, dirs, envi, mntr, ruok, srvr, srst, stat",
		// MonitoringConfig configs
		"metricsProvider.exportJvmInfo": "true",
		"metricsProvider.httpPort":      metricsPort,
		"metricsProvider.className":     "org.apache.zookeeper.metrics.prometheus.PrometheusMetricsProvider",
		// Admin configs
		"admin.enableServer": strconv.FormatBool(enableAdmin),
		"admin.serverPort":   fmt.Sprintf("%d", c.Spec.Ports.Admin),
	}, "clientPort", "secureClientPort", "dataDir", "dataLogDir", "dynamicConfigFile",
		"metricsProvider.httpPort", "admin.enableServer", "admin.serverPort")
	log.Printf("zoo.cfg values: %s\n", str)
	return str
}

// see https://github.com/apache/zookeeper/blob/master/conf/logback.xml
func createZkLogbackXmData(_ *v1alpha1.ZookeeperCluster) string {
	return `
<configuration>
  <property name="zookeeper.console.threshold" value="INFO" />
  <appender name="CONSOLE" class="ch.qos.logback.core.ConsoleAppender">
    <encoder>
      <pattern>%d{ISO8601} [myid:%X{myid}] - %-5p [%t:%C{1}@%L] - %m%n</pattern>
    </encoder>
    <filter class="ch.qos.logback.classic.filter.ThresholdFilter">
      <level>${zookeeper.console.threshold}</level>
    </filter>
  </appender>
  <root level="INFO">
    <appender-ref ref="CONSOLE" />
  </root>
</configuration>
`
}
