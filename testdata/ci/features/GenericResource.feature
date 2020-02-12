#
# 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#

@GenericResource
Feature: Deploy a TestResourceApp application using Alien4Cloud with host generic resources requirements

  Background: Deploying required artifacts
    Given I have built the artifact named "testResource" from templates named "testResource" to Alien
    And I have uploaded the artifact named "testResource" to Alien

  @CI
  @hp
  @cleanupAlien
  Scenario: Deploy a TestResourceApp application using Alien4Cloud with host generic resources requirements
    Given I have created an application named "TestResourceApp" based on template named "org.ystia.ci.tests.test_resource"

    When I deploy the application named "TestResourceApp"

    Then The attribute "gpu" of the instance "0" of the node named "ComputeA" is equal to "gpu2"
    Then The attribute "hostname" of the instance "0" of the node named "ComputeA" is equal to "hostspool-ci-0"

    Then The attribute "gpu" of the instance "0" of the node named "ComputeB" is equal to "gpu0,gpu1"
    Then The attribute "hostname" of the instance "0" of the node named "ComputeB" is equal to "hostspool-ci-1"

