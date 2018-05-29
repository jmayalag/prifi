# PriFi: A Low-Latency, Tracking-Resistant Protocol for Local-Area Anonymity [![Build Status](https://travis-ci.org/lbarman/prifi.svg?branch=master)](https://travis-ci.org/lbarman/prifi)

[back to main README](README.md)


## Deploying PriFi on iOS and Android

We use [go-mobile](https://github.com/golang/mobile) to achieve this goal. This great tool allows us to generate bindings from a Go package that can be later invoked from Java / Kotlin (on Android) and ObjC / Swift (on iOS).

The package `prifiMobile` (found in the folder `prifi-mobile`) contains all the functions, methods and structs that we want to expose to Android and iOS using `go-mobile`.

If you want to expand this package, please be aware of [the type restrictions of go-mobile](https://godoc.org/golang.org/x/mobile/cmd/gobind#hdr-Type_restrictions).

### Install go-mobile

Please check the official [wiki](https://godoc.org/golang.org/x/mobile/cmd/gomobile) for the installation.

Make sure that the target SDK tools are installed before calling `gomobile init`.
- Android: Android SDK and Android NDK are both required.
- iOS: Xcode is required

### Generate a library from prifiMobile for Android

`gomobile bind -target android github.com/lbarman/prifi/prifi-mobile` produces an AAR (Android ARchive) file. The output is named '<package_name>.aar' by default, in our case, it's `prifiMobile.aar`. For more information, pleas check the official [wiki](https://godoc.org/golang.org/x/mobile/cmd/gomobile).

**Note:** _the generated library is big, thus it's not tracked by git._

Normally, the generated AAR can be directly used in any Android apps. Unfortunately in our case, there is one more step to do.

We know that PriFi uses [ONet](https://github.com/dedis/onet) as a network framework, and `ONet` has an OS check at launch. Unfortunately, `ONet.v2` (29 May 2018) currently doesn't support Android, which results an instant crash on Android.

We need to modify the system checking mechanism in `gopkg.in/dedis/onet.v2/cfgpath/cfgpath.go`.
```
func GetConfigPath(appName string) string {
	if len(appName) == 0 {
		log.Panic("appName cannot be empty")
	}

	home := os.Getenv("HOME")
	if home == "" {
		u, err := user.Current()
		if err != nil {
			log.Warn("Could not find the home directory. Switching back to current dir.")
			return getCurrentDir(appName)
		}
		home = u.HomeDir
	}

	switch runtime.GOOS {
	case "darwin":
		return path.Join(home, "Library", "Application Support", appName)
	case "windows":
		return path.Join(os.Getenv("APPDATA"), appName)
	case "linux", "freebsd":
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg != "" {
			return path.Join(xdg, appName)
		}
		return path.Join(home, ".config", appName)
	case "android":
		return path.Join("/data/data/ch.epfl.prifiproxy/files", appName, "config")
	default:
		return getCurrentDir(appName)
	}
}

func GetDataPath(appName string) string {
	switch runtime.GOOS {
	case "linux", "freebsd":
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg != "" {
			return path.Join(xdg, appName)
		}
		return path.Join(os.Getenv("HOME"), ".local", "share", appName)
	case "android":
		return path.Join("/data/data/ch.epfl.prifiproxy/files", appName, "data")
	default:
		p := GetConfigPath(appName)
		return path.Join(p, "data")
	}
}

```
Adding `case "android": p = path.Join("/data/data/ch.epfl.prifiproxy/files", ...)` will solve the problem for our PriFi demo app (package name: `ch.epfl.prifiproxy`). If you want to generate an AAR for your own app, please put the corresponding package name instead of `ch.epfl.prifiproxy`.

### Generate a library from prifiMobile for iOS

We haven't tested on iOS yet. (25 April 2018).


## Link AAR to an Android Studio project

If you use our demo app `prifi-mobile-apps/android/PrifiProxy`, there is nothing to configure, just put the AAR into `.../PrifiProxy/app/libs` and resync the project.

If you want to use the generated AAR in your own app, please put the file in the same location `YourApp/app/libs` and include the following lines into the gradle scripts.

**Project-level build.gradle**

Include
```
flatDir {
  dirs 'libs'
}
```
into
```
allprojects {
  repositories {
    ...
  }
}
```

**App-level build.gradle**

Include
```
implementation(name: 'prifiMobile', ext: 'aar')
```
into
```
dependencies{
  ...
}
```

**Note 1:** _Old gradle versions uses the keyword `compile` instead of `implementation`._

**Note 2:** _If you want to replace AAR with a newer version, please delete the old one and sync gradle, then put the new one in and resync gradle._


## Download our AAR and APK

The steps described above are complicated, so we provide the AAR and the APK of our demo app that we are currently using.

[prifiMobile.aar](https://drive.google.com/file/d/1Pck2us_HcVQHeMkWvHp7w4nR-loVpknZ/view?usp=sharing) (29 May 2018)

[PrifiProxy.apk](https://drive.google.com/file/d/1ABPJ5cSVmpP8_a6U0s-9sjlyM3HqduiE/view?usp=sharing) (29 May 2018)


[back to main README](README.md)
