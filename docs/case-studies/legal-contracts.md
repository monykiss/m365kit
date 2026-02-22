# Case Study: Processing 1,200 Legal Contracts with M365Kit

**Organization:** A mid-size professional services firm
**Problem:** 1,200 contracts in SharePoint, inconsistent naming, unknown external sharing
**Time to solution:** 47 minutes
**M365Kit version:** v1.0.0

---

## The Problem

The firm's legal team had accumulated 1,200+ contract documents across 4 SharePoint
libraries over 5 years. Before a compliance audit, they needed to:

1. Identify which files were shared with external parties
2. Find all contracts containing the old company name ("Meridian Partners")
3. Update all references to the new name ("Atlas Group LLC")
4. Generate a manifest of all contracts for the auditor
5. Extract key parties, dates, and values from each contract

Previously this would have taken 3 days of manual work. With M365Kit: 47 minutes.

---

## Step 1: Audit external sharing (3 minutes)

    export KIT_AZURE_CLIENT_ID="..."
    kit auth login

    kit acl external \
      --site "https://firmsp.sharepoint.com/sites/Legal" \
      --domain "firmname.com" \
      --json > external_shares.json

    # Result: 23 files with external sharing across 11 external parties
    # Previously: 2 hours in SharePoint admin center

---

## Step 2: Find old company name references (8 minutes)

    kit batch './contracts_local/*.docx' \
      --action read \
      --json \
      | jq -r 'select(.content | contains("Meridian Partners")) | .file' \
      > needs_update.txt

    wc -l needs_update.txt
    # 342 contracts contain "Meridian Partners"

---

## Step 3: Bulk rename (6 minutes)

    kit batch './contracts_local/*.docx' \
      --action edit \
      --find "Meridian Partners" \
      --replace "Atlas Group LLC" \
      --in-place \
      --concurrency 8

    # 342 contracts updated in 6 minutes. No Word license required.

---

## Step 4: Generate compliance manifest (4 minutes)

    kit fs manifest ./contracts_local/ --output manifest.json

    # 1,247 contracts
    # Total size: 892.3 MB
    # Date range: 2019-03-14 to 2026-02-20

---

## Step 5: Extract key metadata with AI (26 minutes)

    kit batch './contracts_local/*.docx' \
      --action extract \
      --fields "party_a,party_b,effective_date,contract_value,renewal_date" \
      --json >> contracts_metadata.jsonl

    kit report generate \
      --template legal_summary \
      --data contracts_metadata.jsonl \
      --output compliance_report.docx

---

## Results

| Task | Manual estimate | With M365Kit | Reduction |
|------|----------------|--------------|-----------|
| External sharing audit | 2 hours | 3 minutes | 97% |
| Find old name (1,247 docs) | 4 hours | 8 minutes | 97% |
| Bulk rename (342 docs) | 6 hours | 6 minutes | 98% |
| Generate manifest | 1 hour | 4 minutes | 93% |
| AI metadata extraction | 8 hours | 26 minutes | 95% |
| **Total** | **~21 hours** | **47 minutes** | **96%** |

---

## The Pipeline Version

For teams that run this monthly, a single pipeline automates it:

    name: monthly_contract_audit
    steps:
      - id: acl_audit
        action: acl.audit
        options:
          site: "https://firmsp.sharepoint.com/sites/Legal"
          domain: "firmname.com"

      - id: extract_metadata
        action: ai.extract
        input: "${{ steps.download.output }}"
        options:
          fields: "party_a,party_b,effective_date,contract_value"

      - id: generate_report
        action: report.generate
        input: "${{ steps.extract_metadata.output }}"
        options:
          template: legal_summary

Run: `kit pipeline run monthly_contract_audit.yaml`

---

*Built with M365Kit v1.0.0 by KLYTICS LLC*
