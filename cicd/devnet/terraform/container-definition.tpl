[
  {
    "name": "tfXdcNode",
    "image": "xinfinorg/${xdc_environment}:latest",
    "environment": [
      {"name": "PRIVATE_KEYS", "value": "${private_keys}"}
    ],
    "essential": true,
    "logConfiguration": {
      "logDriver": "awslogs",
      "options": {
        "awslogs-group": "${cloudwatch_group}",
        "awslogs-region": "us-east-1",
        "awslogs-stream-prefix": "ecs"
      }
    },
    "portMappings": [
      {
        "hostPort": 80,
        "protocol": "tcp",
        "containerPort": 80
      },
      {
        "hostPort": 8555,
        "protocol": "tcp",
        "containerPort": 8555
      },
      {
        "hostPort": 8545,
        "protocol": "tcp",
        "containerPort": 8545
      },
      {
        "hostPort": 30304,
        "protocol": "tcp",
        "containerPort": 30304
      }
    ],
    "mountPoints": [
      {
        "containerPath": "/work/xdcchain",
        "sourceVolume": "efs"
      }
    ]
  }
]