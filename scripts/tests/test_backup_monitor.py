import datetime as dt
import hashlib
import importlib.util
import json
import os
import shutil
import subprocess
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('backup_monitor', Path(__file__).resolve().parents[2] / 'deploy/backup/monitor.py')
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)


class BackupMonitorTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.name = 'flagpilot-homolog-20261027T120000Z.dump.gpg'
        (self.root / self.name).write_bytes(b'encrypted fixture')
        (self.root / (self.name + '.sha256')).write_text(hashlib.sha256(b'encrypted fixture').hexdigest() + ' file')
        m.record(self.root, 'success', self.name)
        self.now = dt.datetime.now(dt.timezone.utc)

    def test_fresh_and_stale(self):
        self.assertTrue(m.evaluate(self.root, self.now)[0])
        self.assertFalse(m.evaluate(self.root, self.now + dt.timedelta(hours=27))[0])

    def test_failure_preserves_last_success_and_recovery(self):
        before = (self.root / 'last-success.json').read_bytes()
        m.record(self.root, 'failed')
        self.assertFalse(m.evaluate(self.root)[0])
        self.assertEqual(before, (self.root / 'last-success.json').read_bytes())
        m.record(self.root, 'success', self.name)
        self.assertTrue(m.evaluate(self.root)[0])

    def test_running_and_future_clock(self):
        m.record(self.root, 'running')
        self.assertTrue(m.evaluate(self.root)[0])
        self.assertFalse(m.evaluate(self.root, self.now + dt.timedelta(minutes=31))[0])
        self.assertFalse(m.evaluate(self.root, self.now - dt.timedelta(minutes=10))[0])

    def test_missing_corrupt_and_traversal(self):
        (self.root / self.name).write_bytes(b'corrupted')
        self.assertFalse(m.evaluate(self.root)[0])
        success = json.loads((self.root / 'last-success.json').read_text())
        success['file'] = '../outside'
        m.atomic(self.root, 'last-success.json', success)
        self.assertFalse(m.evaluate(self.root)[0])
        (self.root / 'last-success.json').write_text('bad json')
        self.assertFalse(m.evaluate(self.root)[0])
        (self.root / 'last-success.json').unlink()
        self.assertFalse(m.evaluate(self.root)[0])

    def test_reject_remote_push_and_insecure_permissions(self):
        m.atomic(self.root, 'monitor.json', {'push_url': 'https://example.com/secret'})
        with self.assertRaises(ValueError):
            m.push(self.root, (True, 'OK'))
        (self.root / 'monitor.json').chmod(0o644)
        with self.assertRaises(ValueError):
            m.push(self.root, (True, 'OK'))

    def test_rotation_is_bounded_and_preserves_current_file(self):
        path = self.root / 'pull.log'
        path.write_bytes(b'x' * 1024 * 1024)
        m.rotate(self.root)
        self.assertEqual(path.stat().st_size, 0)
        self.assertEqual((self.root / 'pull.log.1').stat().st_size, 1024 * 1024)

    def test_pull_failure_preserves_previous_backup(self):
        home = self.root / 'home'
        target = home / 'backups/flagpilot/homolog'
        target.mkdir(parents=True)
        source = Path(__file__).resolve().parents[2] / 'deploy/backup'
        shutil.copyfile(source / 'monitor.py', target / 'monitor.py')
        shutil.copyfile(source / 'pull.sh', target / 'pull.sh')
        m.record(target, 'success', self.name)
        before = (target / 'last-success.json').read_bytes()
        commands = self.root / 'bin'
        commands.mkdir()
        ssh = commands / 'ssh'
        ssh.write_text('#!/bin/sh\nexit 1\n')
        ssh.chmod(0o700)
        result = subprocess.run(['bash', str(target / 'pull.sh')], env={**os.environ, 'HOME': str(home), 'PATH': str(commands)+':'+os.environ['PATH']}, capture_output=True)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(m.read(target / 'last-attempt.json')['state'], 'failed')
        self.assertEqual((target / 'last-success.json').read_bytes(), before)
        self.assertEqual(list(target.glob('.incoming.*')), [])
