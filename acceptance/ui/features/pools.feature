@pools
Feature: Resource Pool Management
  In order to use Control Center
  As a CC admin user
  I want to manage resource pools

  Background:
    Given that the admin user is logged in

  Scenario: View default resource pools page
    When I am on the resource pool page
    Then I should see "Resource Pools"
      And I should see the add Resource Pool button
      And I should see "Memory Usage"
      And I should see "Created"
      And I should see "Last Modified"

  Scenario: View Add Resource Pool dialog
    When I am on the resource pool page
      And I click the add Resource Pool button
    Then I should see "Add Resource Pool"
      And I should see "Resource Pool: "
      And I should see the Resource Pool name field
      And I should see "Description: "
      And I should see the Description field

  Scenario: Add a resource pool with a duplicate name
    When I am on the resource pool page
      And I click the add Resource Pool button
      And I fill in the Resource Pool name field with "default"
      And I fill in the Description field with "none"
      And I click "Add Resource Pool"
    Then I should see "Adding pool failed"
      And I should see "Internal Server Error: facade: resource pool exists"

  Scenario: Add a resource pool without specifying a name
    When I am on the resource pool page
      And I click the add Resource Pool button
      And I fill in the Resource Pool name field with ""
      And I fill in the Description field with "none"
      And I click "Add Resource Pool"
    Then I should see "Adding pool failed"
      And I should see "Internal Server Error: empty Kind id"

  Scenario: Add a resource pool
    When I am on the resource pool page
      And I click the add Resource Pool button
      And I fill in the Resource Pool name field with "test"
      And I fill in the Description field with "test"
      And I click "Add Resource Pool"
    Then I should see "Added new Pool"
      And I should see "Added resource pool"
      And I should see an entry for "test" in the table

  Scenario: Delete a resource pool
    When I am on the resource pool page
      And I remove "test"
    Then I should see "This action will permanently delete the resource pool"
    When I click "Remove Pool"
    Then I should see "Removed Pool"
      And I should not see an entry for "test" in the table