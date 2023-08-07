[
  {
    "name": "tfXdcNode",
    "image": "xinfinorg/${xdc_environment}:${image_tag}",
    "environment": [
      {"name": "PRIVATE_KEYS", "value": "${private_keys}"},
      {"name": "LOG_LEVEL", "value": "${log_level}"},
      {"name": "NODE_NAME", "value": "${node_name}"}
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
        "hostPort": 30303,
        "protocol": "tcp",
        "containerPort": 30303
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