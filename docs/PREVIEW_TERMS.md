# DigitalOcean Private Preview Terms for `doctl`

These terms (the "Preview Terms") apply to your participation in the Private
Preview of the next-generation `doctl` experience (the "Preview Offering").

The Preview Terms supplement the DigitalOcean Terms of Service Agreement,
including the Service Terms, or another written agreement governing your use of
DigitalOcean services (the "Agreement"). The Preview Offering is a Beta Service
under the Agreement. Terms not defined here have the meanings given in the
Agreement. If these Preview Terms conflict with the Agreement, these Preview
Terms control only for your use of the Preview Offering.

Participation is voluntary and invite-only. If you do not accept these Preview
Terms, you may not participate.

Invited customers may opt in beginning September 29, 2026.

## 1. Scope and service status

The Preview Offering is a beta version of the `doctl` command-line interface,
distributed through a GitHub release. It tests improvements to errors,
progress, validation, help, authentication configuration, and terminal output.
The Preview Offering is a limited-availability release for testing and
feedback. It is not a General Availability product.

a. **Pre-production use:** The Preview Offering is provided "as is." Do not use
   it for production-critical workloads or automation. Test commands, scripts,
   output parsing, prompts, and exit behavior before relying on them.

b. **No service-level agreement:** No uptime, availability, performance,
   support-response, or compatibility SLA applies to the Preview Offering.

c. **Preview changes:** Commands, flags, output, prompts, defaults, timeouts,
   exit behavior, installation requirements, and supported platforms may change
   during the preview.

d. **Installation:** The Preview Offering must be downloaded manually from the
   approved GitHub release. Homebrew and Snap continue to install the GA
   version of `doctl`.

e. **Account actions:** `doctl` sends requests to DigitalOcean APIs using your
   credentials. Commands may create, modify, or delete resources in your
   DigitalOcean account. You must review commands before running them and use
   appropriately scoped credentials.

## 2. Customer acknowledges it will

a. Use the Preview Offering only for evaluation and testing under the
   Agreement.

b. Keep a stable version of `doctl` available for rollback.

c. Protect API tokens, configuration files, command history, logs, and other
   information stored in its environment.

d. Remove secrets, personal data, and customer content before sending feedback
   or diagnostic information.

e. Test representative workflows before using the Preview Offering with
   important resources or automation.

f. Provide feedback during or at the end of the Private Preview.

## 3. Pricing and billing

a. Access to the Preview Offering is provided AT NO ADDITIONAL CHARGE.

b. Standard charges still apply to DigitalOcean resources and services created
   or used through the Preview Offering.

## 4. Confidentiality and feedback

a. The confidentiality terms in the Agreement apply to the Private Preview.

b. DigitalOcean may use feedback to evaluate and improve its products and
   services, subject to the Agreement.

c. DigitalOcean will not publicly identify you as a participant or use your
   name, logo, or testimonial without separate written approval.

## 5. Support and feedback channels

a. Support is provided on a reasonable-effort basis during standard business
   hours (8:00 AM – 5:00 PM PT, Monday through Friday, excluding holidays).

b. Standard support SLAs do not apply to the Preview Offering.

## 6. Ending the Private Preview

a. You may stop participating at any time by uninstalling the preview version,
   and returning to stable `doctl`.

b. Stopping participation does not delete resources created through `doctl` or
   remove charges for those resources.

c. DigitalOcean does not promise that the Preview Offering will continue to
   Public Preview or General Availability, or that a future version will be
   unchanged.

d. DigitalOcean may continue, extend, change, suspend, or discontinue the
   Preview Offering.

e. If DigitalOcean releases a Public Preview or GA version, these Preview Terms
   will end as stated in the release notice. Continued use will be governed by
   the Agreement and any new terms provided to you.

f. If DigitalOcean discontinues the Preview Offering, DigitalOcean may remove
   preview downloads, documentation, support, and update channels. DigitalOcean
   will endeavor, but is not obligated, to provide advance notice and rollback
   guidance.

## Frequently asked questions

### Why should I participate?

You will get early access to proposed `doctl` improvements and can help shape
the experience before a broader release.

### What is included?

The preview focuses on clearer errors, progress and timeout behavior, earlier
validation, improved help, terminal-aware output, and safer authentication
configuration. It does not include OHS or agent commands or APIs.

### How do I install it?

Download the approved beta from the `doctl` v1.169.0-beta.2 [GitHub
release](https://github.com/digitalocean/doctl/releases). DigitalOcean will
provide installation and rollback instructions. Homebrew and Snap continue to
install GA.

### Can I use it in production automation?

No. Test representative workflows in a safe environment. Stable `doctl` remains
the recommended version for production-critical automation.

### How do I get support?

Support is provided on a best-effort basis through your account team during
standard business hours. Standard support channels and SLAs do not apply.

### How do I leave the preview?

Uninstall the preview version, and follow the provided instructions to return
to stable `doctl`.
