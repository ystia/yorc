#
# 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#

@Job
@openstack
@gcp
@hp
Feature: Deploy a TestJobApp application using Alien4Cloud with a job mock

  Background: Deploying required artifacts
    Given I have uploaded the artifact named "org-ystia-samples-jobmock" to Alien
    And I have built the artifact named "testJob" from templates named "testJob" to Alien
    And I have uploaded the artifact named "testJob" to Alien
    And I have built the artifact named "testJobsSched" from templates named "testJobsSched" to Alien
    And I have uploaded the artifact named "testJobsSched" to Alien

  @CI
  @cleanupAlien
  Scenario: Run a job and check its workflow status
    Given I have created an application named "TestJobApp" based on template named "org.ystia.ci.tests.test_job"
    And I have deployed the application named "TestJobApp"

    When I run the workflow named "run"

    Then The status of the workflow is finally "SUCCEEDED" waiting max "30" seconds

  @CI
  @cleanupAlien
  Scenario: Cancel a job and check its workflow status
    Given I have created an application named "TestJobApp" based on template named "org.ystia.ci.tests.test_job"
    And I have deployed the application named "TestJobApp"

    When I run asynchronously the workflow named "run"
    And I wait for "5" seconds
    And I cancel the last run workflow

    Then The status of the workflow is finally "CANCELLED" waiting max "30" seconds

  @CI
  @cleanupAlien
  Scenario: Run a workload and check its workflow status
    Given I have created an application named "TestJobWorkloadApp" based on template named "org.ystia.ci.tests.test_job_sched"
    And I have deployed the application named "TestJobWorkloadApp"

    When I run the workflow named "run"

    Then The status of the workflow is finally "SUCCEEDED" waiting max "60" seconds
    And The status of the instance "0" of the node named "DelayJob1" is "executed"
    And The status of the instance "0" of the node named "DelayJob21" is "executed"
    And The status of the instance "0" of the node named "DelayJob22" is "executed"
    And The status of the instance "0" of the node named "DelayJob3" is "executed"


