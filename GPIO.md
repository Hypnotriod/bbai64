# Create GPIO overlay example

* Create `/opt/source/dtb-5.10-ti/src/arm64/overlays/BBAI64-GPIO.dts` file with code below:
```txt
/dts-v1/;
/plugin/;

&{/chosen} {
        overlays {
                BBAI64-GPIO = __TIMESTAMP__;
        };
};

&{/} {
        /* Dummy driver to request setup for cape header pins */
        cape_header: pinmux_dummy {
                compatible = "gpio-leds";
                pinctrl-names = "default";
                pinctrl-0 = <
                        /* Update with your header pins to init */
                        &P8_03_default_pin
                        &P8_04_default_pin
                        &P8_05_default_pin
                        &P8_06_default_pin
                >;
        };
};
```

* Compile BBAI64-GPIO.dtbo:
```bash
cd /opt/source/dtb-5.10-ti/
sudo make
```

* Copy `BBAI64-GPIO.dtbo` file to `/boot/firmware/overlays/` folder
```bash
sudo cp /opt/source/dtb-5.10-ti/src/arm64/overlays/BBAI64-GPIO.dtbo /boot/firmware/overlays/
```

* Add `BBAI64-GPIO.dtbo` to `fdtoverlays` in `/boot/firmware/extlinux/extlinux.conf` file and reboot
```txt
    fdtoverlays /overlays/BBAI64-GPIO.dtbo
```
