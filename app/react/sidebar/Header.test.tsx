import { render, screen } from '@testing-library/react';

import { withTestRouter } from '@/react/test-utils/withRouter';

import { Context } from './useSidebarState';
import { Header } from './Header';

function renderComponent(
  props = {},
  sidebarState = { isOpen: true, toggle: vi.fn() }
) {
  const Wrapped = withTestRouter(Header);

  return render(
    <Context.Provider value={sidebarState}>
      <Wrapped {...props} />
    </Context.Provider>
  );
}

describe('Header', () => {
  it('should render 1CRM Developer Console brand when no custom logo', () => {
    renderComponent();

    expect(screen.getByText('1CRM')).toBeInTheDocument();
    expect(screen.getByText('Developer Console')).toBeInTheDocument();
    expect(screen.queryByAltText('Logo')).not.toBeInTheDocument();
  });

  it('should render with custom logo when provided', () => {
    const customLogo = 'https://example.com/custom-logo.png';
    renderComponent({ logo: customLogo });

    const logo = screen.getByAltText('Logo');
    expect(logo).toBeInTheDocument();
    expect(logo).toHaveAttribute('src', customLogo);
  });

  it('should show 1CRM Developer Console caption when sidebar is open and custom logo is provided', () => {
    const customLogo = 'https://example.com/custom-logo.png';
    renderComponent({ logo: customLogo });

    expect(screen.getAllByText('1CRM').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Developer Console').length).toBeGreaterThan(0);
  });

  it('should hide Developer Console subtitle when sidebar is closed', () => {
    renderComponent({}, { isOpen: false, toggle: vi.fn() });

    expect(screen.getByText('1CRM')).toBeInTheDocument();
    expect(screen.queryByText('Developer Console')).not.toBeInTheDocument();
  });

  it('should apply flex-wrap class to logo container', () => {
    renderComponent();

    const logoContainer = screen.getByTestId('portainerSidebar-logoContainer');
    expect(logoContainer).toHaveClass('flex-wrap');
  });

  it('should apply justify-center class when sidebar is closed', () => {
    renderComponent({}, { isOpen: false, toggle: vi.fn() });

    const logoContainer = screen.getByTestId('portainerSidebar-logoContainer');
    expect(logoContainer).toHaveClass('justify-center');
  });

  it('should not apply justify-center class when sidebar is open', () => {
    renderComponent();

    const logoContainer = screen.getByTestId('portainerSidebar-logoContainer');
    expect(logoContainer).not.toHaveClass('justify-center');
  });
});
