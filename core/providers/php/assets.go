package php

import _ "embed"

//go:embed nginx.template.conf
var nginxConf string

//go:embed php-fpm.template.conf
var phpFpmConf string
