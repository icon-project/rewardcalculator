[![Build Status](https://travis-ci.org/icon-project/rewardcalculator.svg?branch=master)](https://travis-ci.org/icon-project/rewardcalculator)
# Reward Calculator

Reward calculator is a daemon which calculates I-Score of ICONists to support IISS.

## Terms
* IISS : The ICON Incentive Scoring System
* I-Score : A metric used to quantify an ICONist’s contribution to the network

For the details, see '[IISS Yellowpaper](https://icon.foundation/download/IISS_Paper_v2.0_EN.pdf)'

## Binaries

### icon_rc
Reward calculator daemon. Do following functions.
* Calculates I-Score
* Claim and Query I-Score
* Communicates with ICON Service via unix domain socket

### rctool
Query debugging information of icon_rc.

## Build
```
# compile binaries
$ make              # for OSX
or
$ make linux        # for linux

# install binaries to system
$ make install
```

## References
 - [ICON Service](https://github.com/icon-project/icon-service)
 - [IISS Yellowpaper](https://icon.foundation/download/IISS_Paper_v2.0_EN.pdf)

## License

This project follows the Apache 2.0 License. Please refer to [LICENSE](https://www.apache.org/licenses/LICENSE-2.0) for details.
