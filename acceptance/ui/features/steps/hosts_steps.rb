When(/^I am on the hosts page$/) do
    visitHostsPage()
end

When(/^I fill in the Host Name field with "(.*?)"$/) do |hostName|
    @hosts_page.hostName_input.set hostName
end

When(/^I fill in the Resource Pool field with "(.*?)"$/) do |resourcePool|
    @hosts_page.resourcePool_input.select resourcePool
end

When(/^I fill in the RAM Commitment field with "(.*?)"$/) do |ramCommitment|
    @hosts_page.ramCommitment_input.set ramCommitment
end

When /^I click the Add-Host button$/ do
    clickAddHostsButton()
end

def visitHostsPage()
    @hosts_page = Hosts.new
    #
    # FIXME: For some reason the following load fails on Chrome for this page,
    #        even though the same syntax works on FF
    # @hosts_page.load
    # expect(@hosts_page).to be_displayed
    within(".navbar-collapse") do
        click_link("Hosts")
    end
    expect(@hosts_page).to be_displayed
end

def clickAddHostsButton()
    @hosts_page.addHosts_button.click
end