# Notes on package maintainer scripts

These scripts originate in the context of a Debian-based system's package
management infrastructure. If this sentence is still in this document and there
are RPM-specific scripts present here, treat this document with intense
skepticism.

## Reference

[Package maintainer scripts and installation procedure](https://www.debian.org/doc/debian-policy/ch-maintainerscripts)

## Initial installation

In the context of a clean install, the following script execution occurs:

```shell
postinstall.sh configure <version>
```

## Removal

In the context of an uninstall, one of the following script executions occurs:

```shell
preremove.sh remove
```

```shell
preremove.sh purge
```

Which one depends on whether the user used `remove` or `purge`.

## Upgrade

In the context of an upgrade from a previous installation, scripts associated
with both the original package and the new package get invoked. This is the
sequence of script executions with their source:

```shell
[old package] preremove.sh upgrade <new version>
[new package] postinstall.sh configure <new version>
```

If the upgrade fails, the following script execution occurs:

```shell
[new package] preremove.sh failed-upgrade <old version>
```
