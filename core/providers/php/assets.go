package php

import _ "embed"

//go:embed nginx.template.conf
var nginxConfTemplateAsset string

//go:embed php-fpm.template.conf
var phpFpmConfTemplateAsset string

//go:embed start-nginx.sh
var startNginxScriptAsset string
