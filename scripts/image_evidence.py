"""Gate conservador de evidências de imagens: inventário obrigatório, sem exceções implícitas."""
from collections import Counter


def summarize(scan, sbom):
    results = scan.get('Results')
    components = sbom.get('components')
    if not isinstance(results, list) or not results or sbom.get('bomFormat') != 'CycloneDX' or not isinstance(components, list) or not components:
        raise RuntimeError('Inventário ausente/inválido; scan não comprova aprovação.')
    counts = Counter()
    findings = []
    for result in results:
        for item in result.get('Vulnerabilities') or []:
            counts[item['Severity']] += 1
            findings.append({key: item.get(key, '') for key in ['VulnerabilityID', 'PkgName', 'InstalledVersion', 'FixedVersion', 'Severity', 'Status', 'PrimaryURL']})
    return {'severityCounts': dict(counts), 'findings': findings, 'sbomFormat': sbom['bomFormat'],
            'sbomComponents': len(components), 'gatePassed': counts['HIGH'] == 0 and counts['CRITICAL'] == 0}
