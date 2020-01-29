#
# 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#

@Placement
Feature: Deploy a testCompute application using Alien4Cloud, apply hostspool placement policies and check runtime

  Background: Deploying required artifacts
    Given I have built the artifact named "testCompute" from templates named "testCompute" to Alien
    And I have uploaded the artifact named "testCompute" to Alien

  @hostspool
  @cleanupAlien
  Scenario: Deploy a testCompute application and do not apply any placement policy
    Given I have created an application named "TestCompute" based on template named "org.ystia.ci.tests.test_compute"

    When I deploy the application named "TestCompute"

    Then The attribute "hostname" of the instance "0" of the node named "Compute" is equal to the attribute "hostname" of the instance "1" of the node named "Compute"

  @hostspool
  @cleanupAlien
  Scenario: Deploy a testCompute application and apply round-robin placement policy
    Given I have created an application named "TestCompute" based on template named "org.ystia.ci.tests.test_compute"
    And I have added a policy named "roundRobinPolicy" of type "yorc.hostspool.policies.RoundRobinPlacement" on targets "Compute"

    When I deploy the application named "TestCompute"

    Then The attribute "hostname" of the instance "0" of the node named "Compute" is different than the attribute "hostname" of the instance "1" of the node named "Compute"

  @hostspool
  @cleanupAlien
  Scenario: Deploy a testCompute application and apply bin-packing placement policy
    Given I have created an application named "TestCompute" based on template named "org.ystia.ci.tests.test_compute"
    And I have added a policy named "binPackingPolicy" of type "yorc.hostspool.policies.BinPackingPlacement" on targets "Compute"

    When I deploy the application named "TestCompute"

    Then The attribute "hostname" of the instance "0" of the node named "Compute" is equal to the attribute "hostname" of the instance "1" of the node named "Compute"

