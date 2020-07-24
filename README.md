# Elevation API

Simple elevation API reading data from SRTM HGT files.

## Supported data

- 30m (1 arc second)
- 90m (3 arc seconds)

## Usage

Build Docker image.

```sh
docker build -t elevation:latest .
```

Start container with mounted directory containing SRTM HGT files.

```sh
docker run --name elevation -v $(pwd)/data:/opt/elevation/data -p 127.0.0.1:8080:8080 -d elevation:latest
```

### Request

Due to limitations in query length, POST method should be used for large requests.

#### GET

Coordinate order is latitude,longitude.

```sh
curl -v 'http://127.0.0.1:8080/json?locations=48.1234,21.1234|49.12,22.12|1,1'
```

#### POST

```sh
curl -v -X POST \
    -H 'Content-Type: application/json'
    -d '{"locations":[{"latitude":48.1234,"longitude":21.1234},{"latitude":49.12,"longitude":22.12},{"latitude":1,"longitude":1}]}' \
    'http://127.0.0.1:8080/json'
```

### Response

- elevation is in meters
- resolution is in arc seconds (1 - 30m, 3 - 90m)
- for location with unknown elevation (missing data file, location out of SRTM range, void) parameter "error" with value 404 will appear in result object.

```json
{
    "result": [
        {
            "latitude": 48.1234,
            "longitude": 21.1234,
            "elevation": 103,
            "resolution": 1
        },
        {
            "latitude": 49.12,
            "longitude": 22.12,
            "elevation": 420,
            "resolution": 1
        },
        {
            "latitude": 1,
            "longitude": 1,
            "error": 404
        }
    ]
}
```

## License

Licensed under MIT license.
