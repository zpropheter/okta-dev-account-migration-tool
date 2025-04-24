# envsync 

> ⚠️ Envsync is currently in ALPHA. Beta release is on track for early May 2025. 

This tool helps you back up and restore configuration from Okta developer accounts. 
It extends the [Okta CLI Client](https://github.com/okta/okta-cli-client) to handle batch operations on all supported API endpoints which may have data to back up. 

## Alternatives

* use [Terraformer](https://github.com/GoogleCloudPlatform/terraformer/blob/master/docs/okta.md) for to capture configuration that you plan to manage in Terraform moving forward
* Consider using [Okta's PowerShell Module](https://github.com/okta/okta-powershell-cli) or the [Okta CLI Client](https://github.com/okta/okta-cli-client)

## Setup

> ⚠️ Setup will change to using the official okta-cli-client when [this PR](https://github.com/okta/okta-cli-client/pull/18) lands

To set up the supported `okta-cli-client`:

```
git clone git@github.com:edunham/okta-cli-client.git
cd okta-cli-client
git checkout file-handling
make install
```
See the `okta-cli-client` README for setup help.

Follow the `okta-cli-client` [configuration instructions](https://github.com/okta/okta-cli-client?tab=readme-ov-file#configuration) to set up access to your developer account. 

## Installation

Clone this repo and install the tool with `make install`

## Usage

```
$ envsync backup
```

## Feedback

Please create an issue on this repo if you have feedback or feature requests!
