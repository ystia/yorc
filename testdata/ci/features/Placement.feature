#
# 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#

@Placement
Feature: Deploy a TestComputeApp application using Alien4Cloud, apply hostspool placement policies and check runtime

  Background: Deploying required artifacts
    Given I have built the artifact named "testCompute" from templates named "testCompute" to Alien
    And I have uploaded the artifact named "testCompute" to Alien

  @CI
  @hp
  @cleanupAlien
  Scenario: Deploy a TestComputeApp application and do not apply any placement policy
    Given I have created an application named "TestComputeApp" based on template named "org.ystia.ci.tests.test_compute"

    When I deploy the application named "TestComputeApp"

    Then The attribute "hostname" of the instance "0" of the node named "Compute" is equal to the attribute "hostname" of the instance "1" of the node named "Compute"

  @CI
  @hp
  @cleanupAlien
  Scenario: Deploy a TestComputeApp application and apply a weight-balanced placement policy
    Given I have created an application named "TestComputeApp" based on template named "org.ystia.ci.tests.test_compute"
    And I have added a policy named "weightBalancedPolicy" of type "yorc.policies.hostspool.WeightBalancedPlacement:1.1.0" on targets "Compute"

    When I deploy the application named "TestComputeApp"

    Then The attribute "hostname" of the instance "0" of the node named "Compute" is different than the attribute "hostname" of the instance "1" of the node named "Compute"

  @CI
  @hp
  @cleanupAlien
  Scenario: Deploy a TestComputeApp application and apply bin-packing placement policy
    Given I have created an application named "TestComputeApp" based on template named "org.ystia.ci.tests.test_compute"
    And I have added a policy named "binPackingPolicy" of type "yorc.policies.hostspool.BinPackingPlacement:1.1.0" on targets "Compute"

    When I deploy the application named "TestComputeApp"

    Then The attribute "hostname" of the instance "0" of the node named "Compute" is equal to the attribute "hostname" of the instance "1" of the node named "Compute"

