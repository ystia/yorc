#
# 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#

@slow
@selenium
@ELK_basic
Feature: Deploy a basic ELK application to Yorc using Alien4Cloud, both set-up using Yorc bootstrap

  Background: Deploying required artifacts
    Given I have uploaded the artifact named "org-ystia-common" to Alien
    And I have uploaded the artifact named "org-ystia-consul-pub" to Alien
    And I have uploaded the artifact named "org-ystia-consul-linux-bash" to Alien
    And I have uploaded the artifact named "org-ystia-java-pub" to Alien
    And I have uploaded the artifact named "org-ystia-java-linux-bash" to Alien
    And I have uploaded the artifact named "org-ystia-elasticsearch-pub" to Alien
    And I have uploaded the artifact named "org-ystia-elasticsearch-linux-bash" to Alien
    And I have uploaded the artifact named "org-ystia-kafka-pub" to Alien
    And I have uploaded the artifact named "org-ystia-kafka-linux-bash" to Alien
    And I have uploaded the artifact named "org-ystia-logstash-pub" to Alien
    And I have uploaded the artifact named "org-ystia-logstash-linux-bash" to Alien
    And I have uploaded the artifact named "org-ystia-kibana-pub" to Alien
    And I have uploaded the artifact named "org-ystia-kibana-linux-bash" to Alien
    And I have uploaded the artifact named "org-ystia-beats-linux-bash" to Alien
    And I have uploaded the artifact named "org-ystia-samples-dummylogs-linux-bash" to Alien
    And I have uploaded the artifact named "org-ystia-samples-topologies-elk_dummylogs" to Alien

  @CI
  @openstack
  @gcp
  @cleanupAlien
  Scenario: Deploy ELK
    Given I have created an application named "TestElkDummyLogs" based on template named "org.ystia.samples.topologies.elk_dummylogs"

    When I deploy the application named "TestElkDummyLogs"
    And I wait for "20" seconds

    Then Kibana contains a dashboard named "DummyLogs"
    And There is data in Kibana
