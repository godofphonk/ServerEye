// Tab functionality for installation section
document.addEventListener('DOMContentLoaded', function() {
    const tabButtons = document.querySelectorAll('.tab-button');
    const tabPanes = document.querySelectorAll('.tab-pane');

    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const tabId = button.getAttribute('data-tab');
            
            // Remove active class from all buttons and panes
            tabButtons.forEach(btn => btn.classList.remove('active'));
            tabPanes.forEach(pane => pane.classList.remove('active'));
            
            // Add active class to clicked button and corresponding pane
            button.classList.add('active');
            document.getElementById(tabId).classList.add('active');
        });
    });

    // Smooth scrolling for navigation links
    const navLinks = document.querySelectorAll('a[href^="#"]');
    navLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const targetId = this.getAttribute('href');
            const targetSection = document.querySelector(targetId);
            
            if (targetSection) {
                const headerHeight = document.querySelector('.header').offsetHeight;
                const targetPosition = targetSection.offsetTop - headerHeight - 20;
                
                window.scrollTo({
                    top: targetPosition,
                    behavior: 'smooth'
                });
            }
        });
    });

    // Add scroll effect to header
    window.addEventListener('scroll', function() {
        const header = document.querySelector('.header');
        if (window.scrollY > 100) {
            header.style.background = 'rgba(255, 255, 255, 0.95)';
            header.style.backdropFilter = 'blur(10px)';
        } else {
            header.style.background = '#fff';
            header.style.backdropFilter = 'none';
        }
    });

    // Animate feature cards on scroll
    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -50px 0px'
    };

    const observer = new IntersectionObserver(function(entries) {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }
        });
    }, observerOptions);

    // Observe feature cards
    const featureCards = document.querySelectorAll('.feature-card');
    featureCards.forEach(card => {
        card.style.opacity = '0';
        card.style.transform = 'translateY(30px)';
        card.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
        observer.observe(card);
    });

    // Copy code functionality
    const codeBlocks = document.querySelectorAll('pre code');
    codeBlocks.forEach(block => {
        const pre = block.parentElement;
        const copyButton = document.createElement('button');
        copyButton.className = 'copy-button';
        copyButton.innerHTML = '<i class="fas fa-copy"></i>';
        copyButton.title = 'Copy to clipboard';
        
        copyButton.addEventListener('click', async () => {
            try {
                await navigator.clipboard.writeText(block.textContent);
                copyButton.innerHTML = '<i class="fas fa-check"></i>';
                copyButton.style.color = '#10b981';
                
                setTimeout(() => {
                    copyButton.innerHTML = '<i class="fas fa-copy"></i>';
                    copyButton.style.color = '';
                }, 2000);
            } catch (err) {
                console.error('Failed to copy text: ', err);
            }
        });
        
        pre.style.position = 'relative';
        pre.appendChild(copyButton);
    });
});

// Add copy button styles
const style = document.createElement('style');
style.textContent = `
    .copy-button {
        position: absolute;
        top: 1rem;
        right: 1rem;
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.2);
        color: #e5e7eb;
        padding: 0.5rem;
        border-radius: 4px;
        cursor: pointer;
        transition: all 0.3s ease;
        font-size: 0.9rem;
    }
    
    .copy-button:hover {
        background: rgba(255, 255, 255, 0.2);
        transform: scale(1.05);
    }
`;
document.head.appendChild(style);
