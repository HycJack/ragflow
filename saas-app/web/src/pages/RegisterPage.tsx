import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useRegister } from '@/hooks/queries';
import { setToken } from '@/api/client';
import { useWorkspace } from '@/store/workspace';
import { ErrorText } from '@/components/Ui';

export default function RegisterPage() {
  const navigate = useNavigate();
  const setProfile = useWorkspace((s) => s.setProfile);
  const [searchParams] = useSearchParams();
  const inviteCode = searchParams.get('invite') ?? '';

  // If a slug-style invite code was provided, accept it too.
  const register = useRegister();
  const [email, setEmail] = useState('');
  const [nickname, setNickname] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    register.mutate(
      { email, nickname, password, invite_code: inviteCode || undefined },
      {
        onSuccess: (res) => {
          setToken(res.token);
          setProfile(res.profile);
          navigate('/onboarding');
        },
        onError: (err) => setError((err as Error).message),
      },
    );
  }

  return (
    <div className="auth-page">
      <div className="card" style={{ width: 400 }}>
        <div className="card-header">
          <h1>Create your account</h1>
          <p>Start building knowledge apps in seconds.</p>
        </div>

        {inviteCode && (
          <div className="badge" style={{ marginBottom: 14 }}>
            Invited — code {inviteCode.slice(0, 10)}…
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="field">
            <label htmlFor="nickname">Name</label>
            <input id="nickname" required value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="Ada Lovelace" />
          </div>
          <div className="field">
            <label htmlFor="email">Email</label>
            <input id="email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" />
          </div>
          <div className="field">
            <label htmlFor="password">Password</label>
            <input id="password" type="password" required minLength={8} value={password} onChange={(e) => setPassword(e.target.value)} placeholder="At least 8 characters" />
          </div>

          {error && <ErrorText>{error}</ErrorText>}

          <button className="btn btn-block" type="submit" disabled={register.isPending} style={{ marginTop: 8 }}>
            {register.isPending ? 'Creating…' : 'Create account'}
          </button>
        </form>

        <p className="muted" style={{ marginTop: 16, fontSize: 14, textAlign: 'center' }}>
          Already have an account? <Link to="/login">Sign in</Link>
        </p>
      </div>
    </div>
  );
}