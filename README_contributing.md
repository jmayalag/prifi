# PriFi: A Low-Latency, Tracking-Resistant Protocol for Local-Area Anonymity [![Build Status](https://travis-ci.org/lbarman/prifi.svg?branch=master)](https://travis-ci.org/lbarman/prifi)

[back to main README](README.md)

## Contributing

This repository uses Travis CI to check continually that the code is bug-free and compliant to coding standards. No one can push to `master` directly.

A typical workflow would be :

```
$ git clone github.com/lbarman/prifi

[do great improvements]

$ git commit -am "I did great changes!"
$ git push
To github.com:lbarman/prifi.git
 ! [remote rejected] master -> master (protected branch hook declined)
```

Your work was rejected, as you are trying to push to master.

```
git checkout -b my-branch
git push -u origin my-branch
```

You can now check in [https://github.com/lbarman/prifi/commits/my-branch](https://github.com/lbarman/prifi/commits/my-branch) that integration tests passed (green check). A yellow dot means that the tests are still running.

Regardless of the result, you can create a new pull request (base: `master`, compare: `my-branch`), and continue commiting changes. When all integration tests passes, you will be able to merge the pull request into master.