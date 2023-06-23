
# powerley-energybridge-homekit

**This project is now archived, see [Status](#status) below.**

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
`powerley_energybridge_instantaneous_demand_watts`.

## Status

This project is now archived, and won't be updated further.

[House Bill 6](https://www.legislature.ohio.gov/legislation/133/hb6) was a law passed by the Ohio General Assembly and signed into law by Governor Mike DeWine in 2019.  It's a real travesty of a law, providing bailouts of failing nuclear and coal power plants owned by a bankrupt division of one of Ohio's largest utility companies paid for by Ohio's utility rate payers.  In order to offset the cost of these bailouts on electricity bills, HB6 also rolled back energy efficiency programs that were funded by rate payers.

It turns out that this law was the result of a [bribery scheme](https://en.wikipedia.org/wiki/Ohio_nuclear_bribery_scandal) involving (among others) Larry Householder, who was Speaker of the Ohio House at the time and a champion of this bill.  Householder was subsequently expelled from the House, but HB6 was never fully repealed.  Householder and Matt Borges were convicted in March 2023 of racketeering conspiracy.  Two other defendants took plea deals and a fifth defendant committed suicide before the trial began.

The whole thing is a real shitshow.  But what does that have to do with this repository?  AEP Ohio provided these Powerley energy bridge devices free of charge to customers as part of its "It's Your Power" energy efficiency program.  Because of HB6's repeal, [AEP Ohio terminated this program](https://ohio-aep.com/ItsYourPower-ProgramEnd) in November 2020.  AEP Ohio's FAQ says, "IT'S YOUR POWER was funded under AEP Ohio's energy efficiency portfolio which has ended. AEP Ohio has not recieved regulatory approval to continue this service."  What "regulatory approval" means here has never been made clear.

Fearing a firmware update that would brick the device, I firewalled mine off from the internet and was able to continue to measure my power usage until May 2023, 2.5 years after the program ended.  At that point, however, the device stopped reporting energy usage.

As I am no longer able to use the device, I am retiring this repository and looking into alternative ways to measure my home electricity usage.

## Installing

The tool can be installed with:

    go install github.com/joeshaw/powerley-energybridge-homekit@latest

Then you can run the service:

    powerley-energybridge-homekit -ip <energybridge ip>

The service will receive messages from the energy bridge every 5
seconds.

The Prometheus exporter runs on port 9525 by default.  This can be
changed with the `-addr` flag.

To pair with HomeKit, open up your Home iOS app, click the + icon,
choose "Add Accessory" and then tap "Don't have a Code or Can't Scan?"
You should see the energy bridge under "Nearby Accessories."  Tap that
and enter the PIN 00102003.

## License

Copyright 2018-2023 Joe Shaw

`powerley-energybridge-homekit` is licensed under the MIT License.
See the LICENSE file for details.


