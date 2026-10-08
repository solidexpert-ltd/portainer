import { baseHref } from '@/portainer/helpers/pathHelper';
import { OAuthStyle } from '@/react/portainer/settings/types';

export default {
  microsoft: {
    authUrl: 'https://login.microsoftonline.com/TENANT_ID/oauth2/v2.0/authorize',
    accessTokenUrl: 'https://login.microsoftonline.com/TENANT_ID/oauth2/v2.0/token',
    resourceUrl: 'https://graph.microsoft.com/v1.0/me',
    logoutUrl: `https://login.microsoftonline.com/TENANT_ID/oauth2/v2.0/logout`,
    userIdentifier: 'userPrincipalName',
    scopes: 'profile openid',
    authStyle: OAuthStyle.InParams,
  },
  google: {
    authUrl: 'https://accounts.google.com/o/oauth2/auth',
    accessTokenUrl: 'https://accounts.google.com/o/oauth2/token',
    resourceUrl: 'https://www.googleapis.com/oauth2/v1/userinfo?alt=json',
    logoutUrl: `https://www.google.com/accounts/Logout?continue=https://appengine.google.com/_ah/logout?continue=${window.location.origin}${baseHref()}#!/auth`,
    userIdentifier: 'email',
    scopes: 'profile email',
    authStyle: OAuthStyle.InParams,
  },
  github: {
    authUrl: 'https://github.com/login/oauth/authorize',
    accessTokenUrl: 'https://github.com/login/oauth/access_token',
    resourceUrl: 'https://api.github.com/user',
    logoutUrl: `https://github.com/logout`,
    userIdentifier: 'login',
    scopes: 'id email name',
    authStyle: OAuthStyle.AutoDetect,
  },
  onecrm: {
    authUrl: 'https://account.1crm.io/api/user/connect/authorize',
    accessTokenUrl: 'https://account.1crm.io/api/user/connect/token',
    resourceUrl: 'https://account.1crm.io/api/user/connect/userinfo',
    logoutUrl: 'https://account.1crm.io/api/user/connect/logout',
    // preferred_username is always set by user-service OidcController (phone-first users
    // often have empty email). GetUsername still falls back to email/phone_number/sub.
    userIdentifier: 'preferred_username',
    // Match OpenIddict permissions for portainer-developer-console (no bare "profile" scope)
    scopes: 'openid user:email user:firstName user:lastName offline_access',
    authStyle: OAuthStyle.InParams,
  },
  custom: { authUrl: '', accessTokenUrl: '', resourceUrl: '', logoutUrl: '', userIdentifier: '', scopes: '', authStyle: OAuthStyle.AutoDetect },
};

export function getProviderByUrl(providerAuthURL = '') {
  if (providerAuthURL.includes('login.microsoftonline.com')) {
    return 'microsoft';
  }

  if (providerAuthURL.includes('accounts.google.com')) {
    return 'google';
  }

  if (providerAuthURL.includes('github.com')) {
    return 'github';
  }

  if (providerAuthURL.includes('account.1crm.io') || providerAuthURL.includes('1crm.io/api/user/connect')) {
    return 'onecrm';
  }

  return 'custom';
}
