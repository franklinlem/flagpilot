"""Regressões do gate: inventário ausente não é evidência de zero vulnerabilidades."""
import sys
from pathlib import Path
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from image_evidence import summarize


class ImageEvidenceTest(unittest.TestCase):
    def test_missing_inventory_fails_closed(self):
        for scan, sbom in [({}, {}), ({'Results': []}, {'bomFormat': 'CycloneDX', 'components': [{'name': 'x'}]}),
                           ({'Results': [{'Target': 'x'}]}, {'bomFormat': 'CycloneDX', 'components': []})]:
            with self.subTest(scan=scan, sbom=sbom), self.assertRaises(RuntimeError):
                summarize(scan, sbom)

    def test_unfixed_high_and_critical_block_even_alongside_low(self):
        vulns = [{'VulnerabilityID': 'CVE-fixture', 'PkgName': 'fixture', 'Severity': level, 'FixedVersion': '', 'Status': 'affected'}
                 for level in ['HIGH', 'CRITICAL', 'LOW']]
        result = summarize({'Results': [{'Vulnerabilities': vulns}]}, {'bomFormat': 'CycloneDX', 'components': [{'name': 'fixture'}]})
        self.assertFalse(result['gatePassed'])
        self.assertEqual(len(result['findings']), 3)
        self.assertEqual(result['severityCounts'], {'HIGH': 1, 'CRITICAL': 1, 'LOW': 1})

    def test_clean_inventory_and_low_only_pass_defined_gate(self):
        for vulns in [None, [], [{'VulnerabilityID': 'CVE-fixture', 'Severity': 'LOW'}]]:
            with self.subTest(vulns=vulns):
                result = summarize({'Results': [{'Vulnerabilities': vulns}]}, {'bomFormat': 'CycloneDX', 'components': [{'name': 'fixture'}]})
                self.assertTrue(result['gatePassed'])


if __name__ == '__main__':
    unittest.main()
