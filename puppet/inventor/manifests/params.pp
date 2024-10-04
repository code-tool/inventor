# == Class inventor::params
#
# default parameters definition
#
class inventor::params {

  $key = 'secret'
  $url = 'NOT-CONFIGURED'
  $cron = '0 */3 * * *'
  $package_ensure = 'installed'
  $config_manage = true
  $manage_package = false
  $package_name = [ 'curl' ]
  $use_defaults = true
  # https://puppet.com/docs/puppet/7/core_facts.html#os
  $info = {}

}
