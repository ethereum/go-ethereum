
# EFS
resource "aws_efs_file_system" "devnet_efs" {
   creation_token = "efs"
   performance_mode = "generalPurpose"
   throughput_mode = "bursting"
   encrypted = "true"
   tags = {
       Name = "TfDevnetEfs"
   }
 }

resource "aws_efs_mount_target" "devnet_efs_efs_mount_target" {
  file_system_id = aws_efs_file_system.devnet_efs.id
  subnet_id      = aws_subnet.devnet_subnet.id
  security_groups = [aws_security_group.devnet_efs_security_group.id]
}

resource "aws_efs_access_point" "devnet_efs_access_point" {
  file_system_id = aws_efs_file_system.devnet_efs.id
  for_each = var.devnet_node_kyes
  root_directory {
    path = "/${each.key}/database"
    creation_info {
      owner_gid = 1001
      owner_uid = 1001
      permissions = 777
    }
  }
  posix_user {
    gid = 1001
    uid = 1001
    secondary_gids = [0]
  }
  
  tags = {
       Name = "TfDevnetEfsAccessPoint-${each.key}"
   }
}