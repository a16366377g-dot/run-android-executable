# Description

This project presents a simple way to run executables on Android.

This is mainly meant to be used in automation apps like Tasker, where we would want to run handy commands like `jq`, `ffmpeg`, etc, many others, natively in Tasker.

&nbsp;

For this, we source the executables from Termux and combine all of them into one standalone executable.

Open Termux if already installed, otherwise get it from [F-Droid - Termux](https://f-droid.org/repo/com.termux_1020.apk).

To build your own flavor -

    git clone --depth 1 https://github.com/HunterXProgrammer/run-android-executable ~/run-android-executable; git -C ~/run-android-executable pull

&nbsp;

    cd ~/run-android-executable

&nbsp;

    ./build.sh jq ffmpeg

(After `./build.sh`, put space seperated list of executables to be packed)

This will generate the standalone executable named `run`.

You can close Termux now. Tap exit from notification.

&nbsp;

Open Tasker to create a Task and select action [Run Shell]. Use this command to add the executable -

```
cp -f /sdcard/run /data/data/net.dinglisch.android.taskerm/files
```

Only need to run this command once every time you build a new flavor.

&nbsp;

After that, you can make `alias` to keep it short in a [Run Shell] action -

```
alias jq="/system/bin/linker$(uname -m | grep -o 64) /data/data/net.dinglisch.android.taskerm/files/run jq"

alias ffmpeg="/system/bin/linker$(uname -m | grep -o 64) /data/data/net.dinglisch.android.taskerm/files/run ffmpeg"

ffmpeg --help

echo '{"Hi":"Hello"}' | jq .Hi
```

Similarly, you can add whatever command you selected when building your own flavor of `run`.

&nbsp;

If you want to try the long way, you can write the entire path every time. Both works, but using `alias` to keep it short as seen above is much easier. If you still want to try -

`/system/bin/linker64 /data/data/net.dinglisch.android.taskerm/files/run jq --help`

Or

`/system/bin/linker64 /data/data/net.dinglisch.android.taskerm/files/run ffmpeg --help`
