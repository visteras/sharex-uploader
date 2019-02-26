# sharex-uploader

If you used nginx-proxy you can run as
```
docker run -d --name sharex --restart always \
                -e 'VIRTUAL_HOST=share.example.com' -e 'VIRTUAL_PORT=3000' \
                -v /home/docker/sharex/images:/app/files \
                visteras/sharex-uploader:latest
```
If not

```
docker run -d --name sharex --restart always \
                -e 'VIRTUAL_HOST=share.example.com' \ 
                -p 80:3000 \
                -v /home/docker/sharex/images:/app/files \
                visteras/sharex-uploader:latest
```
