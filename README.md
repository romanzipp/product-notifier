# Product Notifier

## Build

```
go build -o app .
./app
```

## Configuration

- Copy [.env.example](.env.example) to **.env**
- Update [config.json](config.json)

### `.env`

```dotenv
PUSHOVER_APP_TOKEN=
PUSHOVER_USER_TOKEN=
```

### `config.json`

```json
{
  "loop_interval": 60,
  "products": [
    {
      "title": "Nike Air Force 1 Luxe",
      "img": "https://static.nike.com/a/images/t_PDP_864_v1/f_auto,b_rgb:f5f5f5/076656c4-0ce3-4602-8120-190f8443c67b/air-force-1-luxe-herrenschuh-86CTL1.png",
      "sizes": [
        "39",
        "48.5",
        "49.5"
      ],
      "providers": [
        {
          "id": "nike",
          "url": "https://www.nike.com/de/t/air-force-1-luxe-herrenschuh-86CTL1/DD9605-100"
        }
      ]
    },
    {
      "title": "Nike Air Force 1 High Utility 2.0",
      "img": "https://static.nike.com/a/images/t_PDP_864_v1/f_auto,b_rgb:f5f5f5/f3733d25-3f73-4a96-aebd-625098b25198/air-force-1-high-utility-2-damenschuh-lkNqX7.png",
      "sizes": [
        "36"
      ],
      "providers": [
        {
          "id": "nike",
          "url": "https://www.nike.com/de/t/air-force-1-high-utility-2-damenschuh-lkNqX7"
        }
      ]
    }
  ]
}
```
