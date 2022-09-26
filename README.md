<br/>
<p align="center">
    <a href="https://github.com/ad/domru/blob/master/LICENSE" target="_blank">
        <img src="https://img.shields.io/github/license/ad/domru" alt="GitHub license">
    </a>
    <a href="https://github.com/ad/domru/actions" target="_blank">
        <img src="https://github.com/ad/domru/workflows/Release%20on%20commit%20or%20tag/badge.svg" alt="GitHub actions status">
    </a>
</p>

**ad/domru** is inspired by [alexmorbo/domru](https://github.com/alexmorbo/domru), web server what allows you to control your domofon.

## 🚀&nbsp; Installation and running

```shell
go get -u github.com/ad/domru
```

```shell
domru -login=1234567890 -operator=2 -token=... -refresh=... -port=18000
```

## 🚀&nbsp; Or Docker

```shell
docker build -t ad/domru:latest .
docker run --name domru --rm -p 8080:18000 -e DOMRU_LOGIN=1234567890 -e DOMRU_TOKEN=qwerty -e DOMRU_REFRESH=qwerty -e DOMRU_OPERATOR=2 -e DOMRU_PORT=18000 ad/domru:latest
```

And open in browser [http://localhost:8080/snapshot](http://localhost:8080/snapshot)

## Endpoints and methods

| Endpoint | Method | Description |
| --- | --- | --- |
| `/` | GET | Main interface |
| `/login` | GET/POST | Auth interface |
| `/login/address` | POST | Get address by `phone` and `index` |
| `/sms` | POST | Request sms by `code` |
| `/cameras` | GET | Get list of camera |
| `/door` | GET/POST | Open door by `placeID` and `accessControlID` |
| `/events` | GET | Get list of events |
| `/events/last` | GET | Get last event |
| `/finances` | GET | Get finance info |
| `/operators` | GET | Get operators list |
| `/places` | GET | Get places list |
| `/snapshot` | GET | Get snapshot by `placeID` and `accessControlID` |
| `/stream` | GET | Get link to stream by `cameraID` |

## 🤝&nbsp; Found a bug? Missing a specific feature?

Feel free to **file a new issue** with a respective title and description on the the [ad/domru](https://github.com/ad/domru/issues) repository. If you already found a solution to your problem, **we would love to review your pull request**!

## ✅&nbsp; Requirements

Requires a **Go version higher or equal to 1.11**.

## 📘&nbsp; License

Released under the terms of the [MIT License](LICENSE).
