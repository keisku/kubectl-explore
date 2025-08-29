package explore

func SetDisablePrintPath(o *Options, b bool) {
	o.disablePrintPath = b
}

func SetShowBrackets(o *Options, b bool) {
	o.showBrackets = b
}

func SetAPIVersion(o *Options, apiVersion string) {
	o.apiVersion = apiVersion
}
