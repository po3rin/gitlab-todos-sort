# gitlab-todos-sort

Sort GitLab todos by priority score.

```sh
$ export GITLAB_USER_NAME=<>
$ export GITLAB_HOST=<>
$ export GITLAB_TOKEN=<>

$ gitlab-todos-sort
# url                                score
# https://XXXX/-/merge_requests/1    336.3636363636364
# https://XXXX/-/merge_requests/2    60.97207636814954
```

Open the URL in a browser.

```sh
$ gitlab-todos-sort | awk 'FNR>1' | awk '{print $1}' | xargs open
```

