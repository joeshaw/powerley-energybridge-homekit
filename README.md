
# powerley-energybridge-homecontrol

HomeKit support and Prometheus exporter for Powerley Energy Bridge
devices.

These devices are typically provided by power utility companies under
their brand names.  For instance, AEP Energy Bridge or DTE Energy
Bridge.

This service creates a "sensor" HomeKit accessory and reports energy
usage with characteristics that should be compatible with the [Elgato
Eve](https://itunes.apple.com/us/app/elgato-eve/id917695792?mt=8) app.
However, this functionality isn't quite working at the moment.  (I
suspect the Eve app will only work if the accessory is a "switch"
rather than a "sensor".)

A Prometheus exporter is run (on port 9525 by default) which exports
the current demand in watts under the metric
`powerley_energybridge_instantaneous_demand`.

## Installing

The tool can be installed with:

    go get -u github.com/joeshaw/powerley-energybridge-homecontrol

Then you can run the service:

    powerley-energybridge-homecontrol -ip <energybridge ip>

The service will query the energy bridge every 5 seconds.  This can be
overridden with the `-interval` flag.

The Prometheus exporter runs on port 9525 by default.  This can be
changed with the `-addr` flag.

To pair with HomeKit, open up your Home iOS app, click the + icon,
choose "Add Accessory" and then tap "Don't have a Code or Can't Scan?"
You should see the energy bridge under "Nearby Accessories."  Tap that
and enter the PIN 00102003.

## Contributing

Issues and pull requests are welcome.  When filing a PR, please make
sure the code has been run through `gofmt`.

## License

Copyright 2018 Joe Shaw

`powerley-energybridge-homecontrol` is licensed under the MIT License.
See the LICENSE file for details.


