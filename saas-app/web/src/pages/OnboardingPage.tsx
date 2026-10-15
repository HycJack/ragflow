import { useEffect, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCreateOrg, useOrganizations } from '@/hooks/queries';
import { useWorkspace } from '@/store/workspace';
import { ErrorText, PageLoader } from '@/components/Ui';

export default function OnboardingPage() {
  const navigate = useNavigate();
  const orgsQuery = useOrganizations();
  const createOrg = useCreateOrg();
  const { org, setOrg, setOrganizations } = useWorkspace();

  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [error, setError] = useState('');

  const organizations = orgsQuery.data ?? [];

  // When organizations are loaded and one is already selected, go to workspace.
  useEffect(() => {
    setOrganizations(organizations);
    if (organizations.length > 0 && org) {
      navigate('/workspace/overview', { replace: true });
    }
  }, [organizations, org, navigate, setOrganizations]);

  function selectOrg(id: string) {
    const o = organizations.find((x) => x.id === id);
    if (o) {
      setOrg(o);
      navigate('/workspace/overview', { replace: true });
    }
  }

  function handleCreate(e: FormEvent) {
    e.preventDefault();
    setError('');
    createOrg.mutate(
      { name, slug: slug || undefined },
      {
        onSuccess: (o) => {
          setOrg(o);
          navigate('/workspace/overview', { replace: true });
        },
        onError: (err) => setError((err as Error).message),
      },
    );
  }

  if (orgsQuery.isLoading) {
    return (
      <div className="auth-page">
        <PageLoader />
      </div>
    );
  }

  return (
    <div className="auth-page">
      <div className="card" style={{ width: 440 }}>
        <div className="card-header">
          <h1>Set up your workspace</h1>
          <p>Create an organization or pick an existing one.</p>
        </div>

        {organizations.length > 0 && (
          <>
            <label className="field-title">Your organizations</label>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 20 }}>
              {organizations.map((o) => (
                <button key={o.id} type="button" className="btn btn-secondary" onClick={() => selectOrg(o.id)}>
                  {o.name}
                  <span className="muted" style={{ fontWeight: 400 }}>· {o.my_role}</span>
                </button>
              ))}
            </div>
            <hr className="divider" />
          </>
        )}

        <form onSubmit={handleCreate}>
          <div className="field">
            <label htmlFor="name">Organization name</label>
            <input id="name" required value={name} onChange={(e) => setName(e.target.value)} placeholder="Acme Inc" />
          </div>
          <div className="field">
            <label htmlFor="slug">Workspace slug (optional)</label>
            <input id="slug" value={slug} onChange={(e) => setSlug(e.target.value.toLowerCase())} placeholder="acme" />
          </div>

          {error && <ErrorText>{error}</ErrorText>}

          <button className="btn btn-block" type="submit" disabled={createOrg.isPending} style={{ marginTop: 8 }}>
            {createOrg.isPending ? 'Creating…' : 'Create workspace'}
          </button>
        </form>
      </div>
    </div>
  );
}