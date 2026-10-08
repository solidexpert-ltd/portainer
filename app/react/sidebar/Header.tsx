import clsx from 'clsx';

import { Link } from '@@/Link';

import logoTextOnDark from '@/assets/images/logo/logo-text-on-dark.svg';

import { useSidebarState } from './useSidebarState';
import styles from './Header.module.css';

interface Props {
  logo?: string;
}

export function Header({ logo: customLogo }: Props) {
  const { isOpen } = useSidebarState();

  return (
    <div
      className={clsx('flex w-full flex-wrap', {
        'justify-center pr-5': !isOpen,
      })}
      data-cy="portainerSidebar-logoContainer"
    >
      <Link
        to="portainer.home"
        data-cy="portainerSidebar-homeImage"
        className="text-white no-underline hover:text-white hover:no-underline focus:text-white focus:no-underline focus:outline-none"
      >
        {customLogo ? (
          <img
            src={customLogo}
            className={clsx('img-responsive', styles.logo, {
              '!max-h-[27px]': !isOpen,
            })}
            alt="Logo"
          />
        ) : (
          <span className="flex flex-col leading-tight">
            <img
              src={logoTextOnDark}
              className={clsx(styles.brandMark, {
                [styles.brandMarkCollapsed]: !isOpen,
              })}
              alt="1CRM"
            />
            {isOpen && (
              <span className="mt-1 text-[11px] font-medium tracking-wide text-gray-5">
                Developer Console
              </span>
            )}
          </span>
        )}
      </Link>
      {isOpen && customLogo && (
        <div
          className={clsx(
            'space-x-1 pt-3 text-[9.4px] uppercase tracking-[.28em]',
            'text-gray-3',
            'th-dark:text-gray-warm-6'
          )}
        >
          <span className="font-medium">1CRM</span>
          <span className="font-semibold">Developer Console</span>
        </div>
      )}
    </div>
  );
}
