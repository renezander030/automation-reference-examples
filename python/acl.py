"""Permission revalidation fixture: denied items never reach rendering or counts."""
def retrieve(ids,principal,authorize,load):
    visible=[]
    for id in ids:
        # authorize checks current provider permissions on every request.
        if not authorize(principal,id): continue
        record=load(id)
        # Citation chains are separately authorized. Drop the complete item if any
        # required provenance is unavailable; don't substitute a hidden title.
        if not all(authorize(principal,source) for source in record['sources']): continue
        visible.append(record)
    return {'items':visible,'visible_count':len(visible)}
# Authorization and load require a provider snapshot/transaction to eliminate a
# permission-change race. This fixture models request-time checks, not atomic remote ACLs.
