# == Class inventor::config
#
# creates config files
#
class inventor::config {

  $key              = $inventor::key
  $url              = $inventor::url
  $cron             = $inventor::cron
  $config_manage    = $inventor::config_manage
  $info             = $inventor::info

  if $config_manage {

    file { '/opt/inventor':
      ensure => 'directory',
      owner  => 'root',
      group  => 'root',
      mode   => '0750',
    }->

    file { '/opt/inventor/run.sh':
      ensure  => file,
      owner   => 'root',
      group   => 'root',
      mode    => '0750',
      content => epp("${module_name}/opt/inventor/run.sh.epp", {
        url  => $url,
        key  => $key,
        info => $info,
        }),
    }

    file { '/etc/cron.d/inventor':
      ensure  => file,
      owner   => 'root',
      group   => 'root',
      mode    => '0640',
      content => epp("${module_name}/etc/cron.d/inventor.epp", {
        cron => $cron,
        }),
    }

    $info.each | String $exporter, Hash $data | {
      file { "/opt/inventor/${exporter}.json":
        ensure  => file,
        owner   => 'root',
        group   => 'root',
        mode    => '0640',
        content => epp("${module_name}/opt/inventor/data.json.epp", {
          data => $data,
          }),
      }
    }

  } else {

    $files = [
      '/opt/inventor/*',
      '/etc/cron.d/inventor',
    ]
    file { $files:
      ensure => 'absent',
    }

  }
}
