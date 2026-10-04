# kindle-dashboard-proxy
This project is fundamentally just a drop in cache for the service: https://github.com/sibbl/hass-lovelace-kindle-screensaver. This keeps the last image in memory and fetches it periodically meaning that a home assistant reboot, service failure, or network disconnection won't result in a kindle client getting 404 errors.

That's really all this does. It was written specifically so that I could reboot the home assistant instance for maintenance without fear of breaking something.

## Why
A jailbroken kindle or really any device with an eink display is a really good canidate for rendering a dashboard for things like temperature, but sadly most of them have very limited compute so setting up something like a page scrapper directly on the client is suboptimal at best and even if you could it would not be optimal for something like battery life.

A common solution to this is to deploy something like home assistant with the kiosk extension and have a service running an automation library like puppeter to convert it to a static image. Understandably automation libraries usually are more geared at testing a site and even the best written code and will error. Additionally if you have enough devices the load on something that scrapping an entire page every http request is exceptionally suboptimal from a system resource utilization perspective.

Putting a simple caching service in front of the service doing the page scrapper fixing most of these issues. And this was designed to be run with multiple replicas\stateless so it addresses these problems.

## Build\run
You can use go build or you can edit the Dockerfile to reference the go image of your choice. You will also need to set some environment variables, but the biggest one of note is: `UPSTREAM` needs to be set the URL of your dashboad service.
