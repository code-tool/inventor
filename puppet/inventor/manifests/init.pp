# == Class: inventor
#
# This module installs and configures inventor
#
# === Parameters

class inventor (
  String                        $key             = $inventor::params::key,
  String                        $url             = $inventor::params::url,
  String                        $cron            = $inventor::params::cron,
  Boolean                       $manage_package  = $inventor::params::manage_package,
  Array[String]                 $package_name    = $inventor::params::package_name,
  Boolean                       $config_manage   = $inventor::params::config_manage,
  Hash                          $info            = $inventor::params::info,

  ) inherits inventor::params {

    contain inventor::install
    contain inventor::config

    Class['inventor::install'] ->
    Class['inventor::config']
}
