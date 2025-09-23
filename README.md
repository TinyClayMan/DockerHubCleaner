# DockerHubCleaner
Keep your storage volume small.

This small tool was developed to limit the amount of images or/and their volume present at the same time in Docker Hub.<br />
This is mostly useful if you use basic subscription rates (or a free account), which have an upper limit on storage volume.

It checks whether the container images uploaded to the repository meet the conditions, and if they don't, it deletes the oldest images until the conditions are meet.

# How to run
1. Set env variables for connection to Docker Hub:
- `HUB_REPO_NAME` - The Docker Hub repository for fetching and uploading backups. Format: 'username/repo_name'.
- `HUB_LOGIN` - Your Docker Hub username.
- `HUB_PASSWORD` - Your Docker Hub access token or password.

It’s recommended to save these variables in a local `.env` file and use `source .env` to load them before starting the tool.

2. Set env variables for images limits inside the docker-compose.yml:
- `KEEP_COUNT` - Limit the amount of images that can be present in the repository at once.
- `MAX_SIZE_MB` - Limit the storage volume of the repository that can be used at the same time.

Both, either or neither can be used at the same time. To turn off the parameter, input "-1" as its value.

3. Start the container:
```bash
docker compose up -d
```