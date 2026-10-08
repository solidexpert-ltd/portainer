import { Edit, KeyRound } from 'lucide-react';

import Microsoft from '@/assets/ico/vendor/microsoft.svg?c';
import Google from '@/assets/ico/vendor/google.svg?c';
import Github from '@/assets/ico/vendor/github.svg?c';
import { FeatureId } from '@/react/portainer/feature-flags/enums';

export const options = [
  {
    id: 'microsoft',
    icon: Microsoft,
    label: 'Microsoft',
    description: 'Microsoft OAuth provider',
    value: 'microsoft',
    iconType: 'logo',
    feature: FeatureId.HIDE_INTERNAL_AUTH,
  },
  {
    id: 'google',
    icon: Google,
    label: 'Google',
    description: 'Google OAuth provider',
    value: 'google',
    iconType: 'logo',
    feature: FeatureId.HIDE_INTERNAL_AUTH,
  },
  {
    id: 'github',
    icon: Github,
    label: 'Github',
    description: 'Github OAuth provider',
    value: 'github',
    iconType: 'logo',
    feature: FeatureId.HIDE_INTERNAL_AUTH,
  },
  {
    id: 'onecrm',
    icon: KeyRound,
    iconType: 'badge',
    label: '1CRM (account.1crm.io)',
    description: '1CRM Developer Console via account.1crm.io / user-service',
    value: 'onecrm',
  },
  {
    id: 'custom',
    icon: Edit,
    iconType: 'badge',
    label: 'Custom',
    description: 'Custom OAuth provider',
    value: 'custom',
  },
];
