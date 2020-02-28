#
# 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#

@Monitoring
Feature: Deploy a Welcome application with monitoring to Yorc using Alien4Cloud, both set-up using Yorc bootstrap

  Background: Deploying required artifacts
    Given I have uploaded the artifact named "common" to Alien
    And I have uploaded the artifact named "welcome-linux-bash" to Alien
    And I have uploaded the artifact named "samples-tcp-echo-ansible" to Alien
    And I have uploaded the artifact named "welcome_monitoring" to Alien

  @CI
  @openstack
  @gcp
  @cleanupAlien
  Scenario: Deploy WelcomeMonitoringTest and check TCP and HTTP monitoring
    Given I have created an application named "WelcomeMonitoringTest" based on template named "org.ystia.samples.topologies.welcome_monitoring"

    When I deploy the application named "WelcomeMonitoringTest"
    And I run the workflow named "killTCPEchoServer"
    And I run the workflow named "killWebServer"
    And I wait for "5" seconds

    Then The status of the instance "0" of the node named "MntTCPEcho" is "error"
    And The status of the instance "0" of the node named "MntWelcome" is "error"

  @premium
  @CI
  @openstack
  @gcp
  @cleanupAlien
  Scenario: Update WelcomeMonitoringTest and check TCP and HTTP Monitoring are no longer working
    Given I have created an application named "WelcomeMonitoringTest" based on template named "org.ystia.samples.topologies.welcome_monitoring"
    And I have deployed the application named "WelcomeMonitoringTest"
    And I have deleted the policy named "TCPMonitoring" in the application "WelcomeMonitoringTest"
    And I have deleted the policy named "HTTPMonitoring" in the application "WelcomeMonitoringTest"

    When I update the deployment with application "WelcomeMonitoringTest"
    And I run the workflow named "killTCPEchoServer"
    And I run the workflow named "killWebServer"
    And I wait for "5" seconds

    Then The status of the instance "0" of the node named "MntTCPEcho" is "started"
    And The status of the instance "0" of the node named "MntWelcome" is "started"

