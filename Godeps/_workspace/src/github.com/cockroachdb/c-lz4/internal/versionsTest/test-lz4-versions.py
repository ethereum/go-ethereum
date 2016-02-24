#!/usr/bin/env python3

import glob
import subprocess
import filecmp
import os
import shutil
import sys
import hashlib

repo_url = 'https://github.com/Cyan4973/lz4.git'
tmp_dir_name = 'versionsTest/lz4test'
make_cmd = 'make'
git_cmd = 'git'
test_dat_src = 'README.md'
test_dat = 'test_dat'
head = 'r999'

def proc(cmd_args, pipe=True, dummy=False):
    if dummy:
        return
    if pipe:
        subproc = subprocess.Popen(cmd_args,
                                   stdout=subprocess.PIPE, 
                                   stderr=subprocess.PIPE)
    else:
        subproc = subprocess.Popen(cmd_args)
    return subproc.communicate()

def make(args, pipe=True):
    return proc([make_cmd] + args, pipe)

def git(args, pipe=True):
    return proc([git_cmd] + args, pipe)

def get_git_tags():
    stdout, stderr = git(['tag', '-l', 'r[0-9][0-9][0-9]'])
    tags = stdout.decode('utf-8').split()
    return tags

# http://stackoverflow.com/a/19711609/2132223
def sha1_of_file(filepath):
    with open(filepath, 'rb') as f:
        return hashlib.sha1(f.read()).hexdigest()

if __name__ == '__main__':
    error_code = 0
    base_dir = os.getcwd() + '/..'           # /path/to/lz4
    tmp_dir = base_dir + '/' + tmp_dir_name  # /path/to/lz4/versionsTest/lz4test
    clone_dir = tmp_dir + '/' + 'lz4'        # /path/to/lz4/versionsTest/lz4test/lz4
    programs_dir = base_dir + '/programs'    # /path/to/lz4/programs
    os.makedirs(tmp_dir, exist_ok=True)

    # since Travis clones limited depth, we should clone full repository
    if not os.path.isdir(clone_dir):
        git(['clone', repo_url, clone_dir])

    shutil.copy2(base_dir + '/' + test_dat_src, tmp_dir + '/' + test_dat)

    # Retrieve all release tags
    print('Retrieve all release tags :')
    os.chdir(clone_dir)
    tags = [head] + get_git_tags()
    print(tags);

    # Build all release lz4c and lz4c32
    for tag in tags:
        os.chdir(base_dir)
        dst_lz4c   = '{}/lz4c.{}'  .format(tmp_dir, tag) # /path/to/lz4/test/lz4test/lz4c.<TAG>
        dst_lz4c32 = '{}/lz4c32.{}'.format(tmp_dir, tag) # /path/to/lz4/test/lz4test/lz4c32.<TAG>
        if not os.path.isfile(dst_lz4c) or not os.path.isfile(dst_lz4c32) or tag == head:
            if tag != head:
                r_dir = '{}/{}'.format(tmp_dir, tag)  # /path/to/lz4/test/lz4test/<TAG>
                os.makedirs(r_dir, exist_ok=True)
                os.chdir(clone_dir)
                git(['--work-tree=' + r_dir, 'checkout', tag, '--', '.'], False)
                os.chdir(r_dir + '/programs')  # /path/to/lz4/lz4test/<TAG>/programs
                make(['clean', 'lz4c', 'lz4c32'], False)
            else:
                os.chdir(programs_dir)
                make(['lz4c', 'lz4c32'], False)
            shutil.copy2('lz4c',   dst_lz4c)
            shutil.copy2('lz4c32', dst_lz4c32)

    # Compress test.dat by all released lz4c and lz4c32
    print('Compress test.dat by all released lz4c and lz4c32')
    os.chdir(tmp_dir)
    for lz4 in glob.glob("*.lz4"):
        os.remove(lz4)
    for tag in tags:
        proc(['./lz4c.'   + tag, '-1fz', test_dat, test_dat + '_1_64_' + tag + '.lz4'])
        proc(['./lz4c.'   + tag, '-9fz', test_dat, test_dat + '_9_64_' + tag + '.lz4'])
        proc(['./lz4c32.' + tag, '-1fz', test_dat, test_dat + '_1_32_' + tag + '.lz4'])
        proc(['./lz4c32.' + tag, '-9fz', test_dat, test_dat + '_9_32_' + tag + '.lz4'])

    print('Full list of compressed files')
    lz4s = sorted(glob.glob('*.lz4'))
    for lz4 in lz4s:
        print(lz4 + ' : ' + repr(os.path.getsize(lz4)))

    # Remove duplicated .lz4 files
    print('')
    print('Duplicated files')
    lz4s = sorted(glob.glob('*.lz4'))
    for i, lz4 in enumerate(lz4s):
        if not os.path.isfile(lz4):
            continue
        for j in range(i+1, len(lz4s)):
            lz4t = lz4s[j]
            if not os.path.isfile(lz4t):
                continue
            if filecmp.cmp(lz4, lz4t):
                os.remove(lz4t)
                print('{} == {}'.format(lz4, lz4t))

    print('Enumerate only different compressed files')
    lz4s = sorted(glob.glob('*.lz4'))
    for lz4 in lz4s:
        print(lz4 + ' : ' + repr(os.path.getsize(lz4)) + ', ' + sha1_of_file(lz4))

    # Decompress remained .lz4 files by all released lz4c and lz4c32
    print('Decompression tests and verifications')
    lz4s = sorted(glob.glob('*.lz4'))
    for dec in glob.glob("*.dec"):
        os.remove(dec)
    for lz4 in lz4s:
        print(lz4, end=" ")
        for tag in tags:
            print(tag, end=" ")
            proc(['./lz4c.'   + tag, '-df', lz4, lz4 + '_d64_' + tag + '.dec'])
            proc(['./lz4c32.' + tag, '-df', lz4, lz4 + '_d32_' + tag + '.dec'])
        print(' OK')   # well, here, decompression has worked; but file is not yet verified

    # Compare all '.dec' files with test_dat
    decs = glob.glob('*.dec')
    for dec in decs:
        if not filecmp.cmp(dec, test_dat):
            print('ERR : ' + dec)
            error_code = 1
        else:
            print('OK  : ' + dec)
            os.remove(dec)

    if error_code != 0:
        print('ERROR')

    sys.exit(error_code)
