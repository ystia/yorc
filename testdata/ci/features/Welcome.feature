# file: $GOPATH/yorc-ci-godog/features/Welcome.feature

@Welcome
Feature: Deploy a Welcome application to Yorc using Alien4Cloud, both set-up using Yorc bootstrap

  Background: Deploying required artifacts
    Given I have uploaded the artifact named "common" to Alien
    And I have uploaded the artifact named "welcome-linux-bash" to Alien
    And I have uploaded the artifact named "welcome_basic" to Alien

  @CI
  @openstack
  @gcp
  @cleanupAlien
  Scenario: Deploy Welcome and connect to the home page
    Given I have created an application named "WelcomeTest" based on template named "org.ystia.samples.topologies.welcome_basic"

    When I deploy the application named "WelcomeTest"

    Then The web page with base url stored in attribute "url" on node "Welcome" with path "/" contains a title named "Welcome to Ystia Framework !"

  @premium
  @CI
  @gcp
  @openstack
  @cleanupAlien
  Scenario: Update WelcomeTest by removing/adding a workflow and check this workflow is no longer executable
    Given I have created an application named "WelcomeTest" based on template named "org.ystia.samples.topologies.welcome_basic"
    And I have deployed the application named "WelcomeTest"
    And I have deleted the workflow named "killWebServer" in the application "WelcomeTest"

    When I update the deployment with application "WelcomeTest"
    Then The workflow with name "killWebServer" is no longer executable

    When I create the workflow with name "killWebServer" in the application "WelcomeTest"
    And I add a call-operation activity with operation "stop" on target "Welcome" to the workflow with name "killWebServer" in the application "WelcomeTest"
    And I update the deployment with application "WelcomeTest"
    Then The workflow with name "killWebServer" is again executable
