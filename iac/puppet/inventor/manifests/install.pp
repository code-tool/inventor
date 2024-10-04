# == Class inventor::install
#
# This class is called from inventor for install.
#
class inventor::install {

  $manage_package = $inventor::manage_package
  $package_ensure = $inventor::package_ensure
  $package_name   = $inventor::package_name

  # only do repo management when on a Debian-like system
  if $manage_package {

    include apt

    package { $package_name:
      ensure => $package_ensure,
    }

  }

}
